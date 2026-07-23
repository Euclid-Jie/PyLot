package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"script-manager/internal/db"

	_ "modernc.org/sqlite"
)

func TestWorkflowRunNodeLifecycle(t *testing.T) {
	database := newWorkflowTestDB(t, "workflow-run-node-test")

	now := time.Now()
	if _, err := database.Exec(`
		INSERT INTO scripts(id,name,created_at,updated_at) VALUES
			(1,'准备数据',?,?),
			(2,'同步净值',?,?)`, now, now, now, now); err != nil {
		t.Fatal(err)
	}

	runID, err := createWorkflowRun(9, Graph{Nodes: []Node{
		{ID: "prepare", ScriptID: 1},
		{ID: "sync", ScriptID: 2},
	}})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	var firstName, secondName, firstStatus string
	if err := database.QueryRow(`
		SELECT COUNT(*),MIN(CASE WHEN sort_order=0 THEN script_name END),
			MIN(CASE WHEN sort_order=1 THEN script_name END),
			MIN(CASE WHEN sort_order=0 THEN status END)
		FROM workflow_run_nodes WHERE workflow_run_id=?`, runID).
		Scan(&count, &firstName, &secondName, &firstStatus); err != nil {
		t.Fatal(err)
	}
	if count != 2 || firstName != "准备数据" || secondName != "同步净值" || firstStatus != "pending" {
		t.Fatalf("unexpected node snapshots: count=%d first=%q second=%q status=%q", count, firstName, secondName, firstStatus)
	}

	if err := startWorkflowNode(int(runID), "prepare", 42); err != nil {
		t.Fatal(err)
	}
	finishWorkflowNode(int(runID), "prepare", "success")

	var status string
	var recordID int
	var startedAt, endedAt sql.NullTime
	if err := database.QueryRow(`
		SELECT status,run_record_id,started_at,ended_at
		FROM workflow_run_nodes WHERE workflow_run_id=? AND node_id='prepare'`, runID).
		Scan(&status, &recordID, &startedAt, &endedAt); err != nil {
		t.Fatal(err)
	}
	if status != "success" || recordID != 42 || !startedAt.Valid || !endedAt.Valid {
		t.Fatalf("unexpected completed node: status=%q record=%d started=%v ended=%v", status, recordID, startedAt.Valid, endedAt.Valid)
	}
}

func TestRunPersistsNodeLog(t *testing.T) {
	database := newWorkflowTestDB(t, "workflow-run-log-test")

	cmdPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	if _, err := os.Stat(cmdPath); err != nil {
		t.Skipf("cmd.exe unavailable: %v", err)
	}
	graphJSON, err := json.Marshal(Graph{Nodes: []Node{{ID: "echo", ScriptID: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := database.Exec(`
		INSERT INTO scripts(id,name,work_dir,script_path,launch_mode,created_at,updated_at)
		VALUES(1,'日志测试',?,?,'custom',?,?)`,
		t.TempDir(), cmdPath+" /C echo workflow-node-output", now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
		INSERT INTO workflows(id,name,graph,created_at,updated_at)
		VALUES(7,'日志工作流',?,?,?)`, string(graphJSON), now, now); err != nil {
		t.Fatal(err)
	}

	if err := Run(context.Background(), 7, "", func(int, string, int, string) {}, func(int, string, bool) {}); err != nil {
		t.Fatal(err)
	}

	var workflowStatus, nodeStatus, logOutput string
	var recordID int
	if err := database.QueryRow(`
		SELECT wr.status,wrn.status,wrn.run_record_id,rr.log_output
		FROM workflow_runs wr
		JOIN workflow_run_nodes wrn ON wrn.workflow_run_id=wr.id
		JOIN run_records rr ON rr.id=wrn.run_record_id
		WHERE wr.workflow_id=7`).Scan(&workflowStatus, &nodeStatus, &recordID, &logOutput); err != nil {
		t.Fatal(err)
	}
	if workflowStatus != "success" || nodeStatus != "success" || recordID == 0 || !strings.Contains(logOutput, "workflow-node-output") {
		t.Fatalf("unexpected persisted run: workflow=%q node=%q record=%d log=%q", workflowStatus, nodeStatus, recordID, logOutput)
	}
}

func newWorkflowTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	original := db.DB
	database, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.DB = database
	t.Cleanup(func() {
		database.Close()
		db.DB = original
	})

	if _, err := database.Exec(`
		CREATE TABLE scripts (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			list_id INTEGER,
			interpreter_path TEXT NOT NULL DEFAULT '',
			work_dir TEXT NOT NULL DEFAULT '',
			script_path TEXT NOT NULL DEFAULT '',
			launch_mode TEXT NOT NULL DEFAULT 'script',
			fixed_args TEXT NOT NULL DEFAULT '',
			private_env TEXT NOT NULL DEFAULT '',
			timeout_seconds INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE workflows (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			graph TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE workflow_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_id INTEGER,
			status TEXT,
			started_at DATETIME,
			ended_at DATETIME
		);
		CREATE TABLE workflow_run_nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_run_id INTEGER NOT NULL,
			node_id TEXT NOT NULL,
			script_id INTEGER NOT NULL,
			script_name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			started_at DATETIME,
			ended_at DATETIME,
			run_record_id INTEGER,
			sort_order INTEGER NOT NULL DEFAULT 0,
			UNIQUE(workflow_run_id, node_id)
		);
		CREATE TABLE run_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			script_id INTEGER,
			started_at DATETIME,
			ended_at DATETIME,
			status TEXT,
			log_output TEXT,
			is_error INTEGER DEFAULT 0,
			env_snapshot TEXT,
			created_at DATETIME
		);
		CREATE TABLE running_tasks (
			script_id INTEGER PRIMARY KEY,
			pid INTEGER,
			started_at DATETIME
		);
	`); err != nil {
		t.Fatal(err)
	}
	return database
}
