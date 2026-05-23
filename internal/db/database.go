package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init() error {
	exe, err := os.Executable()
	if err != nil {
		exe, _ = os.Getwd()
	} else {
		exe = filepath.Dir(exe)
	}
	dbPath := filepath.Join(exe, "script-manager.db")
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	return createTables()
}

func createTables() error {
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS global_config (
		id INTEGER PRIMARY KEY,
		env_file_path TEXT,
		updated_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS scripts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
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
	CREATE TABLE IF NOT EXISTS schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id INTEGER,
		cron_expr TEXT,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS run_records (
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
	CREATE TABLE IF NOT EXISTS running_tasks (
		script_id INTEGER PRIMARY KEY,
		pid INTEGER,
		started_at DATETIME
	);
	`)
	return err
}

func CleanOldLogs() error {
	_, err := DB.Exec(`DELETE FROM run_records WHERE created_at < datetime('now', '-7 days')`)
	return err
}
