package script

import (
	"database/sql"
	"testing"

	"script-manager/internal/db"

	_ "modernc.org/sqlite"
)

func TestRunHistoryPreservesSourceAndFiltersWorkflowNodes(t *testing.T) {
	original := db.DB
	database, err := sql.Open("sqlite", "file:runner-source-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.DB = database
	t.Cleanup(func() {
		database.Close()
		db.DB = original
	})

	if _, err := database.Exec(`
		CREATE TABLE run_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			script_id INTEGER,
			started_at DATETIME,
			ended_at DATETIME,
			status TEXT,
			log_output TEXT,
			is_error INTEGER DEFAULT 0,
			env_snapshot TEXT,
			trigger_source TEXT NOT NULL DEFAULT 'unknown',
			schedule_id INTEGER,
			created_at DATETIME
		);`); err != nil {
		t.Fatal(err)
	}

	if _, err := CreateRecord(3, "", db.TriggerSourceManual, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateRecord(3, "", db.TriggerSourceSchedule, 17); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateRecord(3, "", db.TriggerSourceWorkflow, 0); err != nil {
		t.Fatal(err)
	}

	all, err := GetRunHistory(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].TriggerSource != db.TriggerSourceWorkflow || all[1].ScheduleID != 17 {
		t.Fatalf("unexpected complete history: %#v", all)
	}

	topLevel, err := GetTopLevelRunHistory(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(topLevel) != 2 || topLevel[0].TriggerSource != db.TriggerSourceSchedule || topLevel[0].ScheduleID != 17 || topLevel[1].TriggerSource != db.TriggerSourceManual {
		t.Fatalf("unexpected top-level history: %#v", topLevel)
	}
}
