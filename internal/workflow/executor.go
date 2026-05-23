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
	ID       string `json:"id"`
	ScriptID int    `json:"scriptId"`
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

// Run executes a workflow. onStatus is called on each node status change.
func Run(ctx context.Context, workflowID int, globalEnvPath string, onStatus StatusCallback) error {
	var wf db.Workflow
	row := db.DB.QueryRow(`SELECT id,name,graph FROM workflows WHERE id=?`, workflowID)
	if err := row.Scan(&wf.ID, &wf.Name, &wf.Graph); err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	var g Graph
	if err := json.Unmarshal([]byte(wf.Graph), &g); err != nil {
		return fmt.Errorf("invalid graph: %w", err)
	}

	// Create workflow run record
	res, _ := db.DB.Exec(`INSERT INTO workflow_runs(workflow_id,status,started_at) VALUES(?,?,?)`,
		workflowID, "running", time.Now())
	runID, _ := res.LastInsertId()

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
		db.DB.Exec(`UPDATE workflow_runs SET status=?,ended_at=? WHERE id=?`, status, now, runID)
	}

	globalEnv, _ := env.LoadGlobalEnv(globalEnvPath)

	for len(ready) > 0 {
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
				err := runNode(ctx, n, globalEnv, int(runID), nid, onStatus)
				if err != nil {
					layerMu.Lock()
					layerFailed = true
					layerMu.Unlock()
				}
			}(nodeID, node)
		}
		wg.Wait()

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

func runNode(ctx context.Context, n Node, globalEnv map[string]string, runID int, nodeID string, onStatus StatusCallback) error {
	s, err := script.GetByID(n.ScriptID)
	if err != nil {
		return err
	}

	var privateEnv map[string]string
	if s.PrivateEnv != "" {
		json.Unmarshal([]byte(s.PrivateEnv), &privateEnv)
	}
	mergedEnv := env.MergeEnv(globalEnv, privateEnv)
	envSnapshot := env.BuildEnvSnapshot(globalEnv, privateEnv)
	recordID, _ := script.CreateRecord(n.ScriptID, envSnapshot)

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
		OnLog: func(line string, isError bool) {},
		OnStatus: func(status string) {
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
			onStatus(runID, nodeID, n.ScriptID, "timeout")
			select {
			case done <- "timeout":
			default:
			}
		},
	}

	onStatus(runID, nodeID, n.ScriptID, "running")
	if err := script.StartScript(task, int(recordID), cbs); err != nil {
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
		script.StopScript(n.ScriptID)
		return ctx.Err()
	}
}
