package db

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestListRecentRunsIncludesUnscheduledTargetsAndExcludesWorkflowNodes(t *testing.T) {
	original := DB
	database, err := sql.Open("sqlite", "file:recent-runs-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	DB = database
	t.Cleanup(func() {
		database.Close()
		DB = original
	})

	if _, err := database.Exec(`
		CREATE TABLE scripts (id INTEGER PRIMARY KEY,name TEXT NOT NULL);
		CREATE TABLE workflows (id INTEGER PRIMARY KEY,name TEXT NOT NULL);
		CREATE TABLE run_records (
			id INTEGER PRIMARY KEY,
			script_id INTEGER,
			started_at DATETIME,
			ended_at DATETIME,
			status TEXT,
			is_error INTEGER,
			trigger_source TEXT,
			schedule_id INTEGER
		);
		CREATE TABLE workflow_runs (
			id INTEGER PRIMARY KEY,
			workflow_id INTEGER,
			status TEXT,
			started_at DATETIME,
			ended_at DATETIME,
			trigger_source TEXT,
			schedule_id INTEGER
		);
		CREATE TABLE workflow_run_nodes (run_record_id INTEGER);
		INSERT INTO scripts(id,name) VALUES(1,'手动脚本'),(2,'节点脚本');
		INSERT INTO workflows(id,name) VALUES(9,'无定时工作流');
	`); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2026, 7, 23, 14, 0, 0, 0, time.Local)
	if _, err := database.Exec(`
		INSERT INTO run_records(id,script_id,started_at,status,is_error,trigger_source,schedule_id) VALUES
			(1,1,?,'success',0,'manual',NULL),
			(2,2,?,'success',0,'unknown',NULL),
			(3,1,?,'success',0,'schedule',17);
		INSERT INTO workflow_run_nodes(run_record_id) VALUES(2);
	`, base, base.Add(time.Minute), base.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
		INSERT INTO workflow_runs(id,workflow_id,status,started_at,trigger_source,schedule_id)
		VALUES(79,9,'success',?,'manual',NULL)
	`, base.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}

	runs, err := ListRecentRuns(50)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 {
		t.Fatalf("len(runs)=%d, want 3: %#v", len(runs), runs)
	}
	if runs[0].TargetType != "script" || runs[0].TriggerSource != TriggerSourceSchedule || runs[0].ScheduleID != 17 {
		t.Fatalf("unexpected scheduled run: %#v", runs[0])
	}
	if runs[1].RecordID != 79 || runs[1].TargetType != "workflow" || runs[1].TargetName != "无定时工作流" || runs[1].TriggerSource != TriggerSourceManual {
		t.Fatalf("unexpected manual workflow run: %#v", runs[1])
	}
	if runs[2].TargetType != "script" || runs[2].TargetName != "手动脚本" || runs[2].TriggerSource != TriggerSourceManual {
		t.Fatalf("unexpected manual script run: %#v", runs[2])
	}
}
