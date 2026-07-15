package db

import (
	"database/sql"
	"strings"
	"sync"
	"time"
)

var writeMu sync.Mutex

func ExecWrite(query string, args ...any) (sql.Result, error) {
	writeMu.Lock()
	defer writeMu.Unlock()

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		res, err := DB.Exec(query, args...)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if !isSQLiteBusy(err) {
			return nil, err
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	return nil, lastErr
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "busy")
}
