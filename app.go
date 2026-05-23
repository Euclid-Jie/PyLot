package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"script-manager/internal/db"
	"script-manager/internal/env"
	"script-manager/internal/scheduler"
	"script-manager/internal/script"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := db.Init(); err != nil {
		fmt.Println("DB init error:", err)
		return
	}
	db.CleanOldLogs()
	scheduler.Init()
	a.loadSchedules()
	scheduler.Start()
}

func (a *App) shutdown(ctx context.Context) {
	scheduler.Stop()
	for _, id := range script.GetRunningIDs() {
		script.StopScript(id)
	}
	db.DB.Close()
}

func (a *App) loadSchedules() {
	rows, _ := db.DB.Query(`SELECT id,script_id,cron_expr FROM schedules WHERE enabled=1`)
	defer rows.Close()
	for rows.Next() {
		var schedID, scriptID int
		var cronExpr string
		rows.Scan(&schedID, &scriptID, &cronExpr)
		scheduler.AddJob(schedID, scriptID, cronExpr, func(sid int) {
			a.RunScript(sid, "")
		})
	}
}

func (a *App) OpenFileDialog(title string) string {
	path, _ := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	return path
}

func (a *App) OpenDirectoryDialog(title string) string {
	path, _ := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	return path
}

func (a *App) OpenInVSCode(dir string) {
	cmd := exec.Command("code", dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start()
}

type ScriptInferResult struct {
	InterpreterPath string `json:"interpreterPath"`
	WorkDir         string `json:"workDir"`
}

// InferFromScriptPath 根据 .py 文件路径推断虚拟环境解释器和工作目录
func (a *App) InferFromScriptPath(scriptPath string) ScriptInferResult {
	result := ScriptInferResult{WorkDir: filepath.Dir(scriptPath)}

	// 向上查找 .venv 目录
	dir := filepath.Dir(scriptPath)
	for i := 0; i < 5; i++ {
		venvPython := filepath.Join(dir, ".venv", "Scripts", "python.exe")
		if _, err := os.Stat(venvPython); err == nil {
			result.InterpreterPath = venvPython
			result.WorkDir = dir
			return result
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return result
}

func (a *App) GetGlobalConfig() db.GlobalConfig {
	var cfg db.GlobalConfig
	db.DB.QueryRow(`SELECT id,env_file_path,updated_at FROM global_config WHERE id=1`).Scan(&cfg.ID, &cfg.EnvFilePath, &cfg.UpdatedAt)
	return cfg
}

func (a *App) SaveGlobalConfig(cfg db.GlobalConfig) error {
	_, err := db.DB.Exec(`INSERT OR REPLACE INTO global_config(id,env_file_path,updated_at) VALUES(1,?,?)`,
		cfg.EnvFilePath, time.Now())
	return err
}

func (a *App) GetScripts() []db.Script {
	scripts, _ := script.GetAll()
	return scripts
}

func (a *App) GetScriptsByCategory(category string) []db.Script {
	scripts, _ := script.GetByCategory(category)
	return scripts
}

func (a *App) CreateScript(s db.Script) (int, error) {
	id, err := script.Create(s)
	return int(id), err
}

func (a *App) UpdateScript(s db.Script) error {
	return script.Update(s)
}

func (a *App) DeleteScript(id int) error {
	db.DB.Exec(`DELETE FROM schedules WHERE script_id=?`, id)
	return script.Delete(id)
}

func (a *App) RunScript(scriptID int, tempArgs string) error {
	if script.IsRunning(scriptID) {
		return fmt.Errorf("script already running")
	}

	s, err := script.GetByID(scriptID)
	if err != nil {
		return err
	}

	cfg := a.GetGlobalConfig()
	globalEnv, _ := env.LoadGlobalEnv(cfg.EnvFilePath)

	var privateEnv map[string]string
	if s.PrivateEnv != "" {
		json.Unmarshal([]byte(s.PrivateEnv), &privateEnv)
	}

	mergedEnv := env.MergeEnv(globalEnv, privateEnv)
	envSnapshot := env.BuildEnvSnapshot(globalEnv, privateEnv)

	args := s.FixedArgs
	if tempArgs != "" {
		args = tempArgs
	}

	recordID, _ := script.CreateRecord(scriptID, envSnapshot)

	task := script.RunTask{
		ScriptID:        scriptID,
		InterpreterPath: s.InterpreterPath,
		ScriptPath:      s.ScriptPath,
		WorkDir:         s.WorkDir,
		LaunchMode:      s.LaunchMode,
		Args:            args,
		Env:             mergedEnv,
		TimeoutSecs:     s.TimeoutSeconds,
	}

	cbs := script.RunCallbacks{
		OnLog: func(line string, isError bool) {
			runtime.EventsEmit(a.ctx, "log:line", map[string]interface{}{
				"scriptID":  scriptID,
				"line":      line,
				"isError":   isError,
				"timestamp": time.Now().Format("15:04:05"),
			})
		},
		OnStatus: func(status string) {
			runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
				"scriptID": scriptID,
				"status":   status,
			})
			if status == "error" {
				runtime.EventsEmit(a.ctx, "task:alert", map[string]interface{}{
					"scriptID":   scriptID,
					"scriptName": s.Name,
					"reason":     "Script exited with error",
				})
			}
		},
		OnTimeout: func() {
			script.MarkTimeout(int(recordID))
			runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
				"scriptID": scriptID,
				"status":   "timeout",
			})
			runtime.EventsEmit(a.ctx, "task:alert", map[string]interface{}{
				"scriptID":   scriptID,
				"scriptName": s.Name,
				"reason":     "Script timeout - no output",
			})
		},
	}

	runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
		"scriptID": scriptID,
		"status":   "running",
	})

	return script.StartScript(task, int(recordID), cbs)
}

func (a *App) StopScript(scriptID int) error {
	if !script.IsRunning(scriptID) {
		return fmt.Errorf("script not running")
	}
	rec, _ := script.GetLatestRecord(scriptID)
	if rec != nil && rec.Status == "running" {
		script.MarkKilled(rec.ID)
	}
	script.StopScript(scriptID)
	runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
		"scriptID": scriptID,
		"status":   "killed",
	})
	return nil
}

func (a *App) GetRunningScripts() []int {
	return script.GetRunningIDs()
}

func (a *App) GetLatestLog(scriptID int) *db.RunRecord {
	rec, _ := script.GetLatestRecord(scriptID)
	return rec
}

func (a *App) GetRunHistory(scriptID int) []db.RunRecord {
	records, _ := script.GetRunHistory(scriptID)
	return records
}

func (a *App) GetRunDetail(recordID int) *db.RunRecord {
	rec, _ := script.GetRunDetail(recordID)
	return rec
}

func (a *App) GetSchedules() []db.Schedule {
	rows, _ := db.DB.Query(`SELECT id,script_id,cron_expr,enabled,created_at FROM schedules ORDER BY id`)
	defer rows.Close()
	var schedules []db.Schedule
	for rows.Next() {
		var s db.Schedule
		rows.Scan(&s.ID, &s.ScriptID, &s.CronExpr, &s.Enabled, &s.CreatedAt)
		schedules = append(schedules, s)
	}
	return schedules
}

func (a *App) SaveSchedule(s db.Schedule) error {
	if s.ID == 0 {
		res, err := db.DB.Exec(`INSERT INTO schedules(script_id,cron_expr,enabled,created_at) VALUES(?,?,?,?)`,
			s.ScriptID, s.CronExpr, s.Enabled, time.Now())
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		s.ID = int(id)
	} else {
		_, err := db.DB.Exec(`UPDATE schedules SET script_id=?,cron_expr=?,enabled=? WHERE id=?`,
			s.ScriptID, s.CronExpr, s.Enabled, s.ID)
		if err != nil {
			return err
		}
	}

	scheduler.RemoveJob(s.ID)
	if s.Enabled == 1 {
		scheduler.AddJob(s.ID, s.ScriptID, s.CronExpr, func(sid int) {
			a.RunScript(sid, "")
		})
	}
	return nil
}

func (a *App) DeleteSchedule(id int) error {
	scheduler.RemoveJob(id)
	_, err := db.DB.Exec(`DELETE FROM schedules WHERE id=?`, id)
	return err
}

func (a *App) ToggleSchedule(id int, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := db.DB.Exec(`UPDATE schedules SET enabled=? WHERE id=?`, val, id)
	if err != nil {
		return err
	}

	var scriptID int
	var cronExpr string
	db.DB.QueryRow(`SELECT script_id,cron_expr FROM schedules WHERE id=?`, id).Scan(&scriptID, &cronExpr)

	scheduler.RemoveJob(id)
	if enabled {
		scheduler.AddJob(id, scriptID, cronExpr, func(sid int) {
			a.RunScript(sid, "")
		})
	}
	return nil
}

func (a *App) GetScheduleOverview() []scheduler.ScheduleInfo {
	infos := scheduler.GetNextRunTimes()
	for i := range infos {
		s, _ := script.GetByID(infos[i].ScriptID)
		if s != nil {
			infos[i].ScriptName = s.Name
		}
	}
	return infos
}
