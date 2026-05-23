package script

import (
	"bufio"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"script-manager/internal/db"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type RunTask struct {
	ScriptID        int
	InterpreterPath string
	ScriptPath      string
	WorkDir         string
	LaunchMode      string
	Args            string
	Env             []string
	TimeoutSecs     int
}

type RunCallbacks struct {
	OnLog     func(line string, isError bool)
	OnStatus  func(status string)
	OnTimeout func()
}

var (
	runningCmds = map[int]*exec.Cmd{}
	mu          sync.Mutex
)

func StartScript(task RunTask, recordID int, cbs RunCallbacks) error {
	args := []string{}
	scriptArg := task.ScriptPath
	if task.LaunchMode == "module" {
		args = append(args, "-m")
		// strip .py extension and convert path separators to dots
		scriptArg = strings.TrimSuffix(scriptArg, ".py")
		scriptArg = strings.ReplaceAll(scriptArg, "\\", "/")
		// use only the filename stem as module name (python -m runs from WorkDir)
		if idx := strings.LastIndex(scriptArg, "/"); idx >= 0 {
			scriptArg = scriptArg[idx+1:]
		}
	}
	args = append(args, scriptArg)
	if task.Args != "" {
		args = append(args, strings.Fields(task.Args)...)
	}

	cmd := exec.Command(task.InterpreterPath, args...)
	cmd.Dir = task.WorkDir
	cmd.Env = task.Env
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	mu.Lock()
	runningCmds[task.ScriptID] = cmd
	mu.Unlock()

	db.DB.Exec(`INSERT OR REPLACE INTO running_tasks(script_id,pid,started_at) VALUES(?,?,?)`,
		task.ScriptID, cmd.Process.Pid, time.Now())

	lastOutput := time.Now()
	var logMu sync.Mutex

	readLines := func(scanner *bufio.Scanner, isErr bool) {
		for scanner.Scan() {
			line := scanner.Text()
			logMu.Lock()
			lastOutput = time.Now()
			logMu.Unlock()
			cbs.OnLog(line, isErr)
			db.DB.Exec(`UPDATE run_records SET log_output = log_output || ? WHERE id = ?`,
				line+"\n", recordID)
		}
	}

	go readLines(bufio.NewScanner(transform.NewReader(stdout, simplifiedchinese.GBK.NewDecoder())), false)
	go readLines(bufio.NewScanner(transform.NewReader(stderr, simplifiedchinese.GBK.NewDecoder())), true)

	if task.TimeoutSecs > 0 {
		go func() {
			for {
				time.Sleep(10 * time.Second)
				logMu.Lock()
				idle := time.Since(lastOutput)
				logMu.Unlock()
				if idle > time.Duration(task.TimeoutSecs)*time.Second {
					StopScript(task.ScriptID)
					cbs.OnTimeout()
					return
				}
				mu.Lock()
				_, still := runningCmds[task.ScriptID]
				mu.Unlock()
				if !still {
					return
				}
			}
		}()
	}

	go func() {
		err := cmd.Wait()
		mu.Lock()
		delete(runningCmds, task.ScriptID)
		mu.Unlock()

		db.DB.Exec(`DELETE FROM running_tasks WHERE script_id=?`, task.ScriptID)

		status := "success"
		isError := 0
		if err != nil {
			status = "error"
			isError = 1
		}
		now := time.Now()
		db.DB.Exec(`UPDATE run_records SET ended_at=?,status=?,is_error=? WHERE id=?`,
			now, status, isError, recordID)
		cbs.OnStatus(status)
	}()

	return nil
}

func StopScript(scriptID int) {
	mu.Lock()
	cmd, ok := runningCmds[scriptID]
	mu.Unlock()
	if !ok {
		return
	}
	pid := cmd.Process.Pid
	exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", pid)).Run()
}

func IsRunning(scriptID int) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := runningCmds[scriptID]
	return ok
}

func GetRunningIDs() []int {
	mu.Lock()
	defer mu.Unlock()
	ids := make([]int, 0, len(runningCmds))
	for id := range runningCmds {
		ids = append(ids, id)
	}
	return ids
}

func GetPID(scriptID int) int {
	mu.Lock()
	defer mu.Unlock()
	if cmd, ok := runningCmds[scriptID]; ok {
		return cmd.Process.Pid
	}
	return 0
}

func MarkKilled(recordID int) {
	now := time.Now()
	db.DB.Exec(`UPDATE run_records SET ended_at=?,status='killed',is_error=1 WHERE id=?`, now, recordID)
}

func MarkTimeout(recordID int) {
	now := time.Now()
	db.DB.Exec(`UPDATE run_records SET ended_at=?,status='timeout',is_error=1 WHERE id=?`, now, recordID)
}

func CreateRecord(scriptID int, envSnapshot string) (int64, error) {
	res, err := db.DB.Exec(
		`INSERT INTO run_records(script_id,started_at,status,log_output,is_error,env_snapshot,created_at) VALUES(?,?,?,?,?,?,?)`,
		scriptID, time.Now(), "running", "", 0, envSnapshot, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetLatestRecord(scriptID int) (*db.RunRecord, error) {
	row := db.DB.QueryRow(
		`SELECT id,script_id,started_at,ended_at,status,log_output,is_error,env_snapshot,created_at FROM run_records WHERE script_id=? ORDER BY id DESC LIMIT 1`,
		scriptID,
	)
	return scanRecord(row)
}

func GetRunHistory(scriptID int) ([]db.RunRecord, error) {
	rows, err := db.DB.Query(
		`SELECT id,script_id,started_at,ended_at,status,'',is_error,env_snapshot,created_at FROM run_records WHERE script_id=? ORDER BY id DESC LIMIT 20`,
		scriptID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []db.RunRecord
	for rows.Next() {
		r, err := scanRecord(rows)
		if err == nil {
			records = append(records, *r)
		}
	}
	return records, nil
}

func GetRunDetail(recordID int) (*db.RunRecord, error) {
	row := db.DB.QueryRow(
		`SELECT id,script_id,started_at,ended_at,status,log_output,is_error,env_snapshot,created_at FROM run_records WHERE id=?`,
		recordID,
	)
	return scanRecord(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRecord(s scanner) (*db.RunRecord, error) {
	var r db.RunRecord
	var endedAt sql.NullTime
	err := s.Scan(&r.ID, &r.ScriptID, &r.StartedAt, &endedAt, &r.Status, &r.LogOutput, &r.IsError, &r.EnvSnapshot, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	if endedAt.Valid {
		r.EndedAt = &endedAt.Time
	}
	return &r, nil
}
