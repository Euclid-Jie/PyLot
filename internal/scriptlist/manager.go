package scriptlist

import (
	"fmt"
	"strings"
	"time"

	"script-manager/internal/db"
)

func GetAll() ([]db.ScriptList, error) {
	rows, err := db.DB.Query(`
		SELECT l.id,l.name,l.sort_order,COUNT(s.id)
		FROM script_lists l
		LEFT JOIN scripts s ON s.list_id=l.id
		GROUP BY l.id,l.name,l.sort_order
		ORDER BY l.sort_order,l.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []db.ScriptList
	for rows.Next() {
		var item db.ScriptList
		if err := rows.Scan(&item.ID, &item.Name, &item.SortOrder, &item.ScriptCount); err != nil {
			return nil, err
		}
		lists = append(lists, item)
	}
	return lists, rows.Err()
}

func Create(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return 0, err
	}
	var nextOrder int
	if err := db.DB.QueryRow(`SELECT COALESCE(MAX(sort_order),0)+10 FROM script_lists`).Scan(&nextOrder); err != nil {
		return 0, err
	}
	res, err := db.ExecWrite(`INSERT INTO script_lists(name,sort_order,created_at,updated_at) VALUES(?,?,?,?)`, name, nextOrder, time.Now(), time.Now())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return 0, fmt.Errorf("已存在同名列表")
		}
		return 0, err
	}
	return res.LastInsertId()
}

func Rename(id int, name string) error {
	name = strings.TrimSpace(name)
	if id <= 0 {
		return fmt.Errorf("invalid list id")
	}
	if err := validateName(name); err != nil {
		return err
	}
	res, err := db.ExecWrite(`UPDATE script_lists SET name=?,updated_at=? WHERE id=?`, name, time.Now(), id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fmt.Errorf("已存在同名列表")
		}
		return err
	}
	changed, err := res.RowsAffected()
	if err == nil && changed == 0 {
		return fmt.Errorf("script list not found")
	}
	return err
}

func Move(id, direction int) error {
	if id <= 0 || (direction != -1 && direction != 1) {
		return fmt.Errorf("invalid list move")
	}
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentOrder int
	if err := tx.QueryRow(`SELECT sort_order FROM script_lists WHERE id=?`, id).Scan(&currentOrder); err != nil {
		return fmt.Errorf("script list not found")
	}
	operator, order := ">", "ASC"
	if direction < 0 {
		operator, order = "<", "DESC"
	}
	var otherID, otherOrder int
	query := fmt.Sprintf(`SELECT id,sort_order FROM script_lists WHERE sort_order %s ? ORDER BY sort_order %s,id %s LIMIT 1`, operator, order, order)
	if err := tx.QueryRow(query, currentOrder).Scan(&otherID, &otherOrder); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE script_lists SET sort_order=?,updated_at=? WHERE id=?`, otherOrder, time.Now(), id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE script_lists SET sort_order=?,updated_at=? WHERE id=?`, currentOrder, time.Now(), otherID); err != nil {
		return err
	}
	return tx.Commit()
}

func Delete(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid list id")
	}
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE scripts SET list_id=NULL,category='' WHERE list_id=?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM script_lists WHERE id=?`, id)
	if err != nil {
		return err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return fmt.Errorf("script list not found")
	}
	return tx.Commit()
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("请输入列表名称")
	}
	if name == "全部脚本" || name == "未分类" {
		return fmt.Errorf("不能使用系统列表名称")
	}
	return nil
}
