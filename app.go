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
	"script-manager/internal/notify"
	"script-manager/internal/scheduler"
	"script-manager/internal/script"
	"script-manager/internal/workflow"

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
		a.addScheduleJob(schedID, scriptID, cronExpr)
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

func (a *App) GetWindowSize() (int, int) {
	return runtime.WindowGetSize(a.ctx)
}

func (a *App) SetWindowSize(w, h int) {
	runtime.WindowSetSize(a.ctx, w, h)
}

func (a *App) OpenInVSCode(dir string) {
	candidates := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Programs\Microsoft VS Code\bin\code.cmd`),
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft VS Code\bin\code.cmd`),
		filepath.Join(os.Getenv("ProgramW6432"), `Microsoft VS Code\bin\code.cmd`),
	}
	codePath := "code"
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			codePath = p
			break
		}
	}
	cmd := exec.Command("cmd", "/c", codePath, dir)
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
	db.DB.QueryRow(`SELECT id,env_file_path,COALESCE(lark_cli_path,''),COALESCE(lark_open_id,''),updated_at FROM global_config WHERE id=1`).
		Scan(&cfg.ID, &cfg.EnvFilePath, &cfg.LarkCLIPath, &cfg.LarkOpenID, &cfg.UpdatedAt)
	return cfg
}

func (a *App) SaveGlobalConfig(cfg db.GlobalConfig) error {
	_, err := db.DB.Exec(`INSERT OR REPLACE INTO global_config(id,env_file_path,lark_cli_path,lark_open_id,updated_at) VALUES(1,?,?,?,?)`,
		cfg.EnvFilePath, cfg.LarkCLIPath, cfg.LarkOpenID, time.Now())
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
			if status != "running" {
				go notify.Feishu(cfg.LarkCLIPath, cfg.LarkOpenID,
					fmt.Sprintf("[PyLot] %s 执行%s", s.Name, notify.StatusLabel(status)))
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
		a.addScheduleJob(s.ID, s.ScriptID, s.CronExpr)
	}
	return nil
}

func (a *App) addScheduleJob(schedID, scriptID int, cronExpr string) {
	if scriptID < 0 {
		wfID := -scriptID
		scheduler.AddJob(schedID, scriptID, cronExpr, func(_ int) { a.RunWorkflow(wfID) })
	} else {
		scheduler.AddJob(schedID, scriptID, cronExpr, func(sid int) { a.RunScript(sid, "") })
	}
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
		a.addScheduleJob(id, scriptID, cronExpr)
	}
	return nil
}

func (a *App) GetScheduleOverview() []scheduler.ScheduleInfo {
	// Load all schedules from DB (including disabled)
	rows, _ := db.DB.Query(`SELECT id,script_id,cron_expr,enabled FROM schedules ORDER BY id`)
	defer rows.Close()

	nextRuns := map[int]scheduler.ScheduleInfo{}
	for _, info := range scheduler.GetNextRunTimes() {
		nextRuns[info.ScheduleID] = info
	}

	var result []scheduler.ScheduleInfo
	for rows.Next() {
		var schedID, scriptID, enabled int
		var cronExpr string
		rows.Scan(&schedID, &scriptID, &cronExpr, &enabled)
		s, _ := script.GetByID(scriptID)
		name := ""
		if s != nil {
			name = s.Name
		} else if scriptID < 0 {
			wfID := -scriptID
			var wfName string
			db.DB.QueryRow(`SELECT name FROM workflows WHERE id=?`, wfID).Scan(&wfName)
			if wfName != "" {
				name = wfName
			} else {
				name = "工作流"
			}
		}
		info := scheduler.ScheduleInfo{
			ScheduleID: schedID,
			ScriptID:   scriptID,
			ScriptName: name,
			CronExpr:   cronExpr,
			Enabled:    enabled == 1,
		}
		if nr, ok := nextRuns[schedID]; ok {
			info.NextRun = nr.NextRun
		}
		result = append(result, info)
	}
	return result
}

// --- Workflow ---

func (a *App) GetWorkflows() []db.Workflow {
	rows, _ := db.DB.Query(`SELECT id,name,graph,created_at,updated_at FROM workflows ORDER BY id`)
	defer rows.Close()
	var list []db.Workflow
	for rows.Next() {
		var w db.Workflow
		rows.Scan(&w.ID, &w.Name, &w.Graph, &w.CreatedAt, &w.UpdatedAt)
		list = append(list, w)
	}
	return list
}

func (a *App) SaveWorkflow(w db.Workflow) (int, error) {
	now := time.Now()
	if w.ID == 0 {
		res, err := db.DB.Exec(`INSERT INTO workflows(name,graph,created_at,updated_at) VALUES(?,?,?,?)`,
			w.Name, w.Graph, now, now)
		if err != nil {
			return 0, err
		}
		id, _ := res.LastInsertId()
		return int(id), nil
	}
	_, err := db.DB.Exec(`UPDATE workflows SET name=?,graph=?,updated_at=? WHERE id=?`,
		w.Name, w.Graph, now, w.ID)
	return w.ID, err
}

func (a *App) DeleteWorkflow(id int) error {
	_, err := db.DB.Exec(`DELETE FROM workflows WHERE id=?`, id)
	return err
}

func (a *App) RunWorkflow(id int) error {
	cfg := a.GetGlobalConfig()
	var wfName string
	db.DB.QueryRow(`SELECT name FROM workflows WHERE id=?`, id).Scan(&wfName)
	go func() {
		err := workflow.Run(a.ctx, id, cfg.EnvFilePath,
			func(runID int, nodeID string, scriptID int, status string) {
				runtime.EventsEmit(a.ctx, "workflow:node-status", map[string]interface{}{
					"runId": runID, "nodeId": nodeID, "scriptId": scriptID, "status": status,
				})
				runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
					"scriptID": scriptID, "status": status,
				})
			},
			func(scriptID int, line string, isError bool) {
				runtime.EventsEmit(a.ctx, "log:line", map[string]interface{}{
					"scriptID": scriptID, "line": line, "isError": isError,
					"timestamp": time.Now().Format("15:04:05"),
				})
			},
		)
		status := "success"
		if err != nil {
			status = "error"
		}
		runtime.EventsEmit(a.ctx, "workflow:status", map[string]interface{}{
			"workflowId": id,
			"status":     status,
		})
		go notify.Feishu(cfg.LarkCLIPath, cfg.LarkOpenID,
			fmt.Sprintf("[PyLot] 工作流「%s」执行%s", wfName, notify.StatusLabel(status)))
	}()
	return nil
}

func (a *App) GetWorkflowRuns(id int) []db.WorkflowRun {
	rows, _ := db.DB.Query(`SELECT id,workflow_id,status,started_at,ended_at FROM workflow_runs WHERE workflow_id=? ORDER BY id DESC LIMIT 20`, id)
	defer rows.Close()
	var list []db.WorkflowRun
	for rows.Next() {
		var r db.WorkflowRun
		rows.Scan(&r.ID, &r.WorkflowID, &r.Status, &r.StartedAt, &r.EndedAt)
		list = append(list, r)
	}
	return list
}

func (a *App) CopyWorkflow(id int) (int, error) {
	var name, graph string
	db.DB.QueryRow(`SELECT name,graph FROM workflows WHERE id=?`, id).Scan(&name, &graph)
	now := time.Now()
	res, err := db.DB.Exec(`INSERT INTO workflows(name,graph,created_at,updated_at) VALUES(?,?,?,?)`,
		name+" (副本)", graph, now, now)
	if err != nil {
		return 0, err
	}
	newID, _ := res.LastInsertId()
	return int(newID), nil
}
