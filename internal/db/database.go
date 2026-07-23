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
	// WAL mode: concurrent readers+writers without BUSY errors; busy_timeout prevents silent failures
	DB.Exec("PRAGMA journal_mode=WAL")
	DB.Exec("PRAGMA busy_timeout=5000")
	return createTables()
}

func createTables() error {
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS global_config (
		id INTEGER PRIMARY KEY,
		env_file_path TEXT,
		updated_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS scripts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL,
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
	CREATE TABLE IF NOT EXISTS script_lists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL COLLATE NOCASE UNIQUE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		legacy_key TEXT UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
		trigger_source TEXT NOT NULL DEFAULT 'unknown',
		schedule_id INTEGER,
		created_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS running_tasks (
		script_id INTEGER PRIMARY KEY,
		pid INTEGER,
		started_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS workflows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		graph TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME,
		updated_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS workflow_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER,
		status TEXT,
		started_at DATETIME,
		ended_at DATETIME,
		trigger_source TEXT NOT NULL DEFAULT 'unknown',
		schedule_id INTEGER
	);
	CREATE TABLE IF NOT EXISTS workflow_run_nodes (
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
	CREATE INDEX IF NOT EXISTS idx_workflow_run_nodes_run ON workflow_run_nodes(workflow_run_id, sort_order, id);
	CREATE TABLE IF NOT EXISTS services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		command TEXT NOT NULL,
		work_dir TEXT NOT NULL DEFAULT '',
		auto_start INTEGER NOT NULL DEFAULT 0,
		port INTEGER NOT NULL DEFAULT 0,
		protocol TEXT NOT NULL DEFAULT 'http',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`)
	// Migrations: ignore errors if columns already exist
	DB.Exec(`ALTER TABLE global_config ADD COLUMN lark_cli_path TEXT DEFAULT ''`)
	DB.Exec(`ALTER TABLE global_config ADD COLUMN lark_open_id TEXT DEFAULT ''`)
	DB.Exec(`ALTER TABLE services ADD COLUMN port INTEGER NOT NULL DEFAULT 0`)
	DB.Exec(`ALTER TABLE services ADD COLUMN protocol TEXT NOT NULL DEFAULT 'http'`)
	DB.Exec(`ALTER TABLE scripts ADD COLUMN list_id INTEGER`)
	DB.Exec(`ALTER TABLE scripts ADD COLUMN description TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE run_records ADD COLUMN trigger_source TEXT NOT NULL DEFAULT 'unknown'`)
	DB.Exec(`ALTER TABLE run_records ADD COLUMN schedule_id INTEGER`)
	DB.Exec(`ALTER TABLE workflow_runs ADD COLUMN trigger_source TEXT NOT NULL DEFAULT 'unknown'`)
	DB.Exec(`ALTER TABLE workflow_runs ADD COLUMN schedule_id INTEGER`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_run_records_script_source ON run_records(script_id,trigger_source,id DESC)`)
	if err := migrateScriptLists(); err != nil {
		return err
	}
	return err
}

func migrateScriptLists() error {
	var applied int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name='script_lists_v1'`).Scan(&applied); err != nil {
		return err
	}
	if applied > 0 {
		return nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`
		INSERT OR IGNORE INTO script_lists(name, sort_order, legacy_key) VALUES
			('数据爬取上传', 10, 'crawler'),
			('数据处理', 20, 'processor'),
			('个人工具', 30, 'tool');

		INSERT OR IGNORE INTO script_lists(name, sort_order, legacy_key)
		SELECT category, 100 + ROW_NUMBER() OVER (ORDER BY category), category
		FROM scripts
		WHERE TRIM(category) <> ''
		GROUP BY category;

		UPDATE scripts
		SET list_id = (
			SELECT id FROM script_lists WHERE legacy_key = scripts.category
		)
		WHERE list_id IS NULL AND TRIM(category) <> '';

		UPDATE scripts SET category='' WHERE list_id IS NOT NULL;
	`); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO schema_migrations(name) VALUES('script_lists_v1')`); err != nil {
		return err
	}
	return tx.Commit()
}

func CleanOldLogs() error {
	_, err := ExecWrite(`DELETE FROM run_records WHERE created_at < datetime('now', '-7 days')`)
	return err
}
