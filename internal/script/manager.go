package script

import (
	"database/sql"
	"time"

	"script-manager/internal/db"
)

func GetAll() ([]db.Script, error) {
	rows, err := db.DB.Query(`SELECT id,name,category,interpreter_path,work_dir,script_path,launch_mode,fixed_args,private_env,timeout_seconds,created_at,updated_at FROM scripts ORDER BY category,name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scripts []db.Script
	for rows.Next() {
		s, err := scanScript(rows)
		if err == nil {
			scripts = append(scripts, *s)
		}
	}
	return scripts, nil
}

func GetByCategory(category string) ([]db.Script, error) {
	rows, err := db.DB.Query(`SELECT id,name,category,interpreter_path,work_dir,script_path,launch_mode,fixed_args,private_env,timeout_seconds,created_at,updated_at FROM scripts WHERE category=? ORDER BY name`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scripts []db.Script
	for rows.Next() {
		s, err := scanScript(rows)
		if err == nil {
			scripts = append(scripts, *s)
		}
	}
	return scripts, nil
}

func GetByID(id int) (*db.Script, error) {
	row := db.DB.QueryRow(`SELECT id,name,category,interpreter_path,work_dir,script_path,launch_mode,fixed_args,private_env,timeout_seconds,created_at,updated_at FROM scripts WHERE id=?`, id)
	return scanScript(row)
}

func Create(s db.Script) (int64, error) {
	now := time.Now()
	res, err := db.DB.Exec(
		`INSERT INTO scripts(name,category,interpreter_path,work_dir,script_path,launch_mode,fixed_args,private_env,timeout_seconds,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		s.Name, s.Category, s.InterpreterPath, s.WorkDir, s.ScriptPath, s.LaunchMode, s.FixedArgs, s.PrivateEnv, s.TimeoutSeconds, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func Update(s db.Script) error {
	_, err := db.DB.Exec(
		`UPDATE scripts SET name=?,category=?,interpreter_path=?,work_dir=?,script_path=?,launch_mode=?,fixed_args=?,private_env=?,timeout_seconds=?,updated_at=? WHERE id=?`,
		s.Name, s.Category, s.InterpreterPath, s.WorkDir, s.ScriptPath, s.LaunchMode, s.FixedArgs, s.PrivateEnv, s.TimeoutSeconds, time.Now(), s.ID,
	)
	return err
}

func Delete(id int) error {
	_, err := db.DB.Exec(`DELETE FROM scripts WHERE id=?`, id)
	return err
}

type scriptScanner interface {
	Scan(dest ...any) error
}

func scanScript(s scriptScanner) (*db.Script, error) {
	var sc db.Script
	var createdAt, updatedAt sql.NullTime
	err := s.Scan(&sc.ID, &sc.Name, &sc.Category, &sc.InterpreterPath, &sc.WorkDir, &sc.ScriptPath, &sc.LaunchMode, &sc.FixedArgs, &sc.PrivateEnv, &sc.TimeoutSeconds, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		sc.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		sc.UpdatedAt = updatedAt.Time
	}
	return &sc, nil
}
