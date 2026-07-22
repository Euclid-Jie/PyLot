package script

import (
	"database/sql"
	"time"

	"script-manager/internal/db"
)

func GetAll() ([]db.Script, error) {
	rows, err := db.DB.Query(`SELECT s.id,s.name,COALESCE(s.description,''),s.category,COALESCE(s.list_id,0),s.interpreter_path,s.work_dir,s.script_path,s.launch_mode,s.fixed_args,s.private_env,s.timeout_seconds,s.created_at,s.updated_at FROM scripts s LEFT JOIN script_lists l ON l.id=s.list_id ORDER BY COALESCE(l.sort_order,2147483647),s.name`)
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
	rows, err := db.DB.Query(`SELECT id,name,COALESCE(description,''),category,COALESCE(list_id,0),interpreter_path,work_dir,script_path,launch_mode,fixed_args,private_env,timeout_seconds,created_at,updated_at FROM scripts WHERE category=? OR list_id=(SELECT id FROM script_lists WHERE legacy_key=?) ORDER BY name`, category, category)
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
	row := db.DB.QueryRow(`SELECT id,name,COALESCE(description,''),category,COALESCE(list_id,0),interpreter_path,work_dir,script_path,launch_mode,fixed_args,private_env,timeout_seconds,created_at,updated_at FROM scripts WHERE id=?`, id)
	return scanScript(row)
}

func Create(s db.Script) (int64, error) {
	now := time.Now()
	res, err := db.ExecWrite(
		`INSERT INTO scripts(name,description,category,list_id,interpreter_path,work_dir,script_path,launch_mode,fixed_args,private_env,timeout_seconds,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.Name, s.Description, s.Category, nullableListID(s.ListID), s.InterpreterPath, s.WorkDir, s.ScriptPath, s.LaunchMode, s.FixedArgs, s.PrivateEnv, s.TimeoutSeconds, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func Update(s db.Script) error {
	_, err := db.ExecWrite(
		`UPDATE scripts SET name=?,description=?,category=?,list_id=?,interpreter_path=?,work_dir=?,script_path=?,launch_mode=?,fixed_args=?,private_env=?,timeout_seconds=?,updated_at=? WHERE id=?`,
		s.Name, s.Description, s.Category, nullableListID(s.ListID), s.InterpreterPath, s.WorkDir, s.ScriptPath, s.LaunchMode, s.FixedArgs, s.PrivateEnv, s.TimeoutSeconds, time.Now(), s.ID,
	)
	return err
}

func Delete(id int) error {
	_, err := db.ExecWrite(`DELETE FROM scripts WHERE id=?`, id)
	return err
}

type scriptScanner interface {
	Scan(dest ...any) error
}

func scanScript(s scriptScanner) (*db.Script, error) {
	var sc db.Script
	var createdAt, updatedAt sql.NullTime
	err := s.Scan(&sc.ID, &sc.Name, &sc.Description, &sc.Category, &sc.ListID, &sc.InterpreterPath, &sc.WorkDir, &sc.ScriptPath, &sc.LaunchMode, &sc.FixedArgs, &sc.PrivateEnv, &sc.TimeoutSeconds, &createdAt, &updatedAt)
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

func nullableListID(id int) any {
	if id <= 0 {
		return nil
	}
	return id
}
