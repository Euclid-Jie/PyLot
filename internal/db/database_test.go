package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestScriptListMigrationRunsOnce(t *testing.T) {
	original := DB
	database, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	DB = database
	t.Cleanup(func() {
		database.Close()
		DB = original
	})

	if _, err := DB.Exec(`
		CREATE TABLE scripts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category TEXT NOT NULL
		);
		INSERT INTO scripts(name,category) VALUES
			('Crawler','crawler'),
			('Processor','processor'),
			('Tool','tool');
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
		INSERT INTO run_records(script_id,status,created_at) VALUES(1,'success',CURRENT_TIMESTAMP);
		CREATE TABLE workflow_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_id INTEGER,
			status TEXT,
			started_at DATETIME,
			ended_at DATETIME
		);
		INSERT INTO workflow_runs(workflow_id,status,started_at) VALUES(1,'success',CURRENT_TIMESTAMP);
	`); err != nil {
		t.Fatal(err)
	}
	if err := createTables(); err != nil {
		t.Fatal(err)
	}
	if _, err := DB.Exec(`UPDATE scripts SET description='旧脚本说明' WHERE name='Crawler'`); err != nil {
		t.Fatalf("description migration failed: %v", err)
	}
	var description string
	if err := DB.QueryRow(`SELECT description FROM scripts WHERE name='Crawler'`).Scan(&description); err != nil {
		t.Fatal(err)
	}
	if description != "旧脚本说明" {
		t.Fatalf("description = %q", description)
	}

	var runSource, workflowSource string
	var runScheduleID, workflowScheduleID sql.NullInt64
	if err := DB.QueryRow(`SELECT trigger_source,schedule_id FROM run_records WHERE id=1`).Scan(&runSource, &runScheduleID); err != nil {
		t.Fatalf("run source migration failed: %v", err)
	}
	if err := DB.QueryRow(`SELECT trigger_source,schedule_id FROM workflow_runs WHERE id=1`).Scan(&workflowSource, &workflowScheduleID); err != nil {
		t.Fatalf("workflow source migration failed: %v", err)
	}
	if runSource != TriggerSourceUnknown || workflowSource != TriggerSourceUnknown || runScheduleID.Valid || workflowScheduleID.Valid {
		t.Fatalf("unexpected migrated sources: run=%q/%v workflow=%q/%v", runSource, runScheduleID, workflowSource, workflowScheduleID)
	}

	var assigned, cleared int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM scripts WHERE list_id IS NOT NULL`).Scan(&assigned); err != nil {
		t.Fatal(err)
	}
	if err := DB.QueryRow(`SELECT COUNT(*) FROM scripts WHERE category=''`).Scan(&cleared); err != nil {
		t.Fatal(err)
	}
	if assigned != 3 || cleared != 3 {
		t.Fatalf("migration assigned=%d cleared=%d, want 3 and 3", assigned, cleared)
	}

	if _, err := DB.Exec(`DELETE FROM script_lists WHERE name='个人工具'`); err != nil {
		t.Fatal(err)
	}
	if err := createTables(); err != nil {
		t.Fatal(err)
	}
	var recreated int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM script_lists WHERE name='个人工具'`).Scan(&recreated); err != nil {
		t.Fatal(err)
	}
	if recreated != 0 {
		t.Fatal("deleted default list was recreated after migration had completed")
	}
}
