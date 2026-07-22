package script

import (
	"database/sql"
	"testing"

	"script-manager/internal/db"

	_ "modernc.org/sqlite"
)

func TestScriptDescriptionPersists(t *testing.T) {
	original := db.DB
	database, err := sql.Open("sqlite", "file:script-description-test?mode=memory&cache=shared")
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			list_id INTEGER,
			interpreter_path TEXT,
			work_dir TEXT,
			script_path TEXT,
			launch_mode TEXT,
			fixed_args TEXT,
			private_env TEXT,
			timeout_seconds INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE script_lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sort_order INTEGER NOT NULL DEFAULT 0
		);
	`); err != nil {
		t.Fatal(err)
	}

	id, err := Create(db.Script{
		Name:            "净值同步",
		Description:     "同步每日净值并上传",
		InterpreterPath: "python",
		ScriptPath:      "sync.py",
		LaunchMode:      "script",
	})
	if err != nil {
		t.Fatal(err)
	}

	script, err := GetByID(int(id))
	if err != nil {
		t.Fatal(err)
	}
	if script.Description != "同步每日净值并上传" {
		t.Fatalf("description = %q", script.Description)
	}

	script.Description = "同步并校验每日净值"
	if err := Update(*script); err != nil {
		t.Fatal(err)
	}
	updated, err := GetByID(int(id))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Description != "同步并校验每日净值" {
		t.Fatalf("updated description = %q", updated.Description)
	}
}
