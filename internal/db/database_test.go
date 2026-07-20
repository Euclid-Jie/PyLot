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
	`); err != nil {
		t.Fatal(err)
	}
	if err := createTables(); err != nil {
		t.Fatal(err)
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
