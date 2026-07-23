package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"script-manager/internal/db"
	"script-manager/internal/env"
	"script-manager/internal/script"
)

type Node struct {
	ID       string  `json:"id"`
	ScriptID int     `json:"scriptId"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type StatusCallback func(workflowRunID int, nodeID string, scriptID int, status string)
type LogCallback func(scriptID int, line string, isError bool)

func ParseGraph(raw string) (Graph, error) {
	var g Graph
	if err := json.Unmarshal([]byte(raw), &g); err != nil {
		return g, fmt.Errorf("invalid graph: %w", err)
	}
	if len(g.Nodes) == 0 {
		return g, fmt.Errorf("workflow requires at least one node")
	}

	nodeIDs := make(map[string]struct{}, len(g.Nodes))
	inDegree := make(map[string]int, len(g.Nodes))
	successors := make(map[string][]string, len(g.Nodes))
	for _, node := range g.Nodes {
		if node.ID == "" {
			return g, fmt.Errorf("workflow contains a node without an id")
		}
		if _, exists := nodeIDs[node.ID]; exists {
			return g, fmt.Errorf("workflow contains duplicate node id %s", node.ID)
		}
		if _, err := script.GetByID(node.ScriptID); err != nil {
			return g, fmt.Errorf("workflow node %s references a missing script", node.ID)
		}
		nodeIDs[node.ID] = struct{}{}
		inDegree[node.ID] = 0
	}
	for _, edge := range g.Edges {
		if _, ok := nodeIDs[edge.Source]; !ok {
			return g, fmt.Errorf("workflow edge references missing source node %s", edge.Source)
		}
		if _, ok := nodeIDs[edge.Target]; !ok {
			return g, fmt.Errorf("workflow edge references missing target node %s", edge.Target)
		}
		successors[edge.Source] = append(successors[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	ready := make([]string, 0, len(g.Nodes))
	for id, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	visited := 0
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		visited++
		for _, successor := range successors[id] {
			inDegree[successor]--
			if inDegree[successor] == 0 {
				ready = append(ready, successor)
			}
		}
	}
	if visited != len(g.Nodes) {
		return g, fmt.Errorf("workflow graph contains a cycle")
	}
	return g, nil
}

// Run executes a workflow. onStatus is called on each node status change.
func Run(ctx context.Context, workflowID int, globalEnvPath string, onStatus StatusCallback, onLog LogCallback) error {
	var wf db.Workflow
	row := db.DB.QueryRow(`SELECT id,name,graph FROM workflows WHERE id=?`, workflowID)
	if err := row.Scan(&wf.ID, &wf.Name, &wf.Graph); err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	g, err := ParseGraph(wf.Graph)
	if err != nil {
		return err
	}

	runID, err := createWorkflowRun(workflowID, g)
	if err != nil {
		return fmt.Errorf("create workflow run: %w", err)
	}

	// Build adjacency: node id -> list of successor node ids
	// Also build in-degree map
	nodeMap := map[string]Node{}
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}
	successors := map[string][]string{}
	inDegree := map[string]int{}
	for _, n := range g.Nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range g.Edges {
		successors[e.Source] = append(successors[e.Source], e.Target)
		inDegree[e.Target]++
	}

	// Kahn's topological sort — process layer by layer
	ready := []string{}
	for id, deg := range inDegree {
		if deg == 0 {
			ready = append(ready, id)
		}
	}

	var mu sync.Mutex
	failed := false

	finishWorkflow := func(status string) {
		now := time.Now()
		if status != "success" {
			db.ExecWrite(`UPDATE workflow_run_nodes SET status='skipped' WHERE workflow_run_id=? AND status='pending'`, runID)
		}
		db.ExecWrite(`UPDATE workflow_runs SET status=?,ended_at=? WHERE id=?`, status, now, runID)
	}

	globalEnv, _ := env.LoadGlobalEnv(globalEnvPath)

	for len(ready) > 0 {
		if err := ctx.Err(); err != nil {
			finishWorkflow("killed")
			return err
		}
		mu.Lock()
		if failed {
			mu.Unlock()
			break
		}
		mu.Unlock()

		// Run current layer concurrently
		var wg sync.WaitGroup
		layerFailed := false
		var layerMu sync.Mutex

		for _, nodeID := range ready {
			node := nodeMap[nodeID]
			wg.Add(1)
			go func(nid string, n Node) {
				defer wg.Done()
				err := runNode(ctx, n, globalEnv, int(runID), nid, onStatus, onLog)
				if err != nil {
					layerMu.Lock()
					layerFailed = true
					layerMu.Unlock()
				}
			}(nodeID, node)
		}
		wg.Wait()
		if err := ctx.Err(); err != nil {
			finishWorkflow("killed")
			return err
		}

		if layerFailed {
			mu.Lock()
			failed = true
			mu.Unlock()
			break
		}

		// Advance to next layer
		next := []string{}
		for _, nid := range ready {
			for _, succ := range successors[nid] {
				inDegree[succ]--
				if inDegree[succ] == 0 {
					next = append(next, succ)
				}
			}
		}
		ready = next
	}

	if failed {
		finishWorkflow("error")
		return fmt.Errorf("workflow failed")
	}
	finishWorkflow("success")
	return nil
}

func runNode(ctx context.Context, n Node, globalEnv map[string]string, runID int, nodeID string, onStatus StatusCallback, onLog LogCallback) error {
	s, err := script.GetByID(n.ScriptID)
	if err != nil {
		finishWorkflowNode(runID, nodeID, "error")
		onStatus(runID, nodeID, n.ScriptID, "error")
		return err
	}

	var privateEnv map[string]string
	if s.PrivateEnv != "" {
		json.Unmarshal([]byte(s.PrivateEnv), &privateEnv)
	}
	mergedEnv := env.MergeEnv(globalEnv, privateEnv)
	envSnapshot := env.BuildEnvSnapshot(globalEnv, privateEnv)
	recordID, err := script.CreateRecord(n.ScriptID, envSnapshot)
	if err != nil {
		finishWorkflowNode(runID, nodeID, "error")
		onStatus(runID, nodeID, n.ScriptID, "error")
		return err
	}
	if err := startWorkflowNode(runID, nodeID, int(recordID)); err != nil {
		script.MarkError(int(recordID))
		finishWorkflowNode(runID, nodeID, "error")
		onStatus(runID, nodeID, n.ScriptID, "error")
		return err
	}

	done := make(chan string, 1)

	task := script.RunTask{
		ScriptID:        n.ScriptID,
		InterpreterPath: s.InterpreterPath,
		ScriptPath:      s.ScriptPath,
		WorkDir:         s.WorkDir,
		LaunchMode:      s.LaunchMode,
		Args:            s.FixedArgs,
		Env:             mergedEnv,
		TimeoutSecs:     s.TimeoutSeconds,
	}

	cbs := script.RunCallbacks{
		OnLog: func(line string, isError bool) { onLog(n.ScriptID, line, isError) },
		OnStatus: func(status string) {
			if status != "running" {
				finishWorkflowNode(runID, nodeID, status)
			}
			onStatus(runID, nodeID, n.ScriptID, status)
			if status != "running" {
				select {
				case done <- status:
				default:
				}
			}
		},
		OnTimeout: func() {
			script.MarkTimeout(int(recordID))
			finishWorkflowNode(runID, nodeID, "timeout")
			onStatus(runID, nodeID, n.ScriptID, "timeout")
			select {
			case done <- "timeout":
			default:
			}
		},
	}

	onStatus(runID, nodeID, n.ScriptID, "running")
	if err := script.StartScript(task, int(recordID), cbs); err != nil {
		script.MarkError(int(recordID))
		finishWorkflowNode(runID, nodeID, "error")
		onStatus(runID, nodeID, n.ScriptID, "error")
		return err
	}

	// Wait for completion or context cancellation
	select {
	case finalStatus := <-done:
		if finalStatus == "error" || finalStatus == "timeout" || finalStatus == "killed" {
			return fmt.Errorf("node %s failed: %s", nodeID, finalStatus)
		}
		return nil
	case <-ctx.Done():
		script.MarkKilled(int(recordID))
		script.StopScript(n.ScriptID)
		finishWorkflowNode(runID, nodeID, "killed")
		onStatus(runID, nodeID, n.ScriptID, "killed")
		return ctx.Err()
	}
}

func createWorkflowRun(workflowID int, graph Graph) (int64, error) {
	scriptNames := make([]string, len(graph.Nodes))
	for i, node := range graph.Nodes {
		s, err := script.GetByID(node.ScriptID)
		if err != nil {
			return 0, err
		}
		scriptNames[i] = s.Name
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO workflow_runs(workflow_id,status,started_at) VALUES(?,?,?)`,
		workflowID, "running", time.Now())
	if err != nil {
		return 0, err
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for i, node := range graph.Nodes {
		if _, err := tx.Exec(`
			INSERT INTO workflow_run_nodes(workflow_run_id,node_id,script_id,script_name,status,sort_order)
			VALUES(?,?,?,?,?,?)`, runID, node.ID, node.ScriptID, scriptNames[i], "pending", i); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return runID, nil
}

func startWorkflowNode(runID int, nodeID string, recordID int) error {
	_, err := db.ExecWrite(`
		UPDATE workflow_run_nodes
		SET status='running',started_at=?,run_record_id=?
		WHERE workflow_run_id=? AND node_id=? AND status='pending'`,
		time.Now(), recordID, runID, nodeID)
	return err
}

func finishWorkflowNode(runID int, nodeID string, status string) {
	db.ExecWrite(`
		UPDATE workflow_run_nodes
		SET status=?,ended_at=?
		WHERE workflow_run_id=? AND node_id=? AND status IN ('pending','running')`,
		status, time.Now(), runID, nodeID)
}
