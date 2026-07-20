package scriptlist

import (
	"database/sql"
	"testing"

	"script-manager/internal/db"

	_ "modernc.org/sqlite"
)

func TestDeleteMovesScriptsToUnassigned(t *testing.T) {
	original := db.DB
	database, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.DB = database
	t.Cleanup(func() {
		database.Close()
		db.DB = original
	})
	if _, err := db.DB.Exec(`
		CREATE TABLE script_lists (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE scripts (id INTEGER PRIMARY KEY, category TEXT NOT NULL, list_id INTEGER);
		INSERT INTO script_lists(id,name,sort_order) VALUES(7,'临时列表',10);
		INSERT INTO scripts(id,category,list_id) VALUES(3,'legacy',7);
	`); err != nil {
		t.Fatal(err)
	}

	if err := Delete(7); err != nil {
		t.Fatal(err)
	}
	var listID sql.NullInt64
	var category string
	if err := db.DB.QueryRow(`SELECT list_id,category FROM scripts WHERE id=3`).Scan(&listID, &category); err != nil {
		t.Fatal(err)
	}
	if listID.Valid || category != "" {
		t.Fatalf("script remained assigned: list=%v category=%q", listID, category)
	}
}

func TestMoveSwapsListOrder(t *testing.T) {
	original := db.DB
	database, err := sql.Open("sqlite", "file:script-list-move?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.DB = database
	t.Cleanup(func() {
		database.Close()
		db.DB = original
	})
	if _, err := db.DB.Exec(`
		CREATE TABLE script_lists (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL, created_at DATETIME, updated_at DATETIME);
		INSERT INTO script_lists(id,name,sort_order) VALUES(1,'A',10),(2,'B',20),(3,'C',30);
	`); err != nil {
		t.Fatal(err)
	}
	if err := Move(2, -1); err != nil {
		t.Fatal(err)
	}
	rows, err := db.DB.Query(`SELECT name FROM script_lists ORDER BY sort_order`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var order []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		order = append(order, name)
	}
	if len(order) != 3 || order[0] != "B" || order[1] != "A" || order[2] != "C" {
		t.Fatalf("order=%v, want [B A C]", order)
	}
}
