package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"script-manager/internal/db"
	"script-manager/internal/env"
	"script-manager/internal/notify"
	"script-manager/internal/scheduler"
	"script-manager/internal/script"
	svc "script-manager/internal/service"
	"script-manager/internal/workflow"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx             context.Context
	workflowMu      sync.Mutex
	workflowCancels map[int]context.CancelFunc
}

func NewApp() *App {
	return &App{workflowCancels: make(map[int]context.CancelFunc)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := db.Init(); err != nil {
		fmt.Println("DB init error:", err)
		return
	}
	db.CleanOldLogs()
	if err := script.CleanupStaleRuns(); err != nil {
		fmt.Println("Stale run cleanup error:", err)
	}
	scheduler.Init()
	a.loadSchedules()
	scheduler.Start()
	a.autoStartServices()
}

func (a *App) shutdown(ctx context.Context) {
	scheduler.Stop()
	a.workflowMu.Lock()
	for _, cancel := range a.workflowCancels {
		cancel()
	}
	a.workflowMu.Unlock()
	for _, id := range script.GetRunningIDs() {
		script.StopScript(id)
	}
	if err := script.CleanupStaleRuns(); err != nil {
		fmt.Println("Stale run cleanup error:", err)
	}
	svc.StopAll()
	db.DB.Close()
}

func (a *App) loadSchedules() {
	rows, _ := db.DB.Query(`SELECT id,script_id,cron_expr FROM schedules WHERE enabled=1`)
	defer rows.Close()
	for rows.Next() {
		var schedID, scriptID int
		var cronExpr string
		rows.Scan(&schedID, &scriptID, &cronExpr)
		if err := a.addScheduleJob(schedID, scriptID, cronExpr); err != nil {
			fmt.Println("Schedule load error:", schedID, err)
		}
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

func (a *App) OpenInVSCode(dir string) error {
	dir, err := existingDirectory(dir)
	if err != nil {
		return err
	}
	candidates := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Programs\Microsoft VS Code\bin\code.cmd`),
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft VS Code\bin\code.cmd`),
		filepath.Join(os.Getenv("ProgramW6432"), `Microsoft VS Code\bin\code.cmd`),
	}
	codePath := ""
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			codePath = p
			break
		}
	}
	if codePath == "" {
		codePath, err = exec.LookPath("code")
		if err != nil {
			return fmt.Errorf("未找到 VS Code 命令，请确认已安装 VS Code 并将 code 加入 PATH")
		}
	}
	cmd := exec.Command("cmd", "/c", codePath, dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func (a *App) OpenInFileExplorer(dir string) error {
	dir, err := existingDirectory(dir)
	if err != nil {
		return err
	}
	cmd := exec.Command("explorer.exe", dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func existingDirectory(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("工作目录不能为空")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("解析工作目录失败: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("工作目录不存在: %s", absDir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("工作目录不是文件夹: %s", absDir)
	}
	return absDir, nil
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

func (a *App) GetScript(id int) (*db.Script, error) {
	return script.GetByID(id)
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
	wfRows, err := db.DB.Query(`SELECT name,graph FROM workflows`)
	if err != nil {
		return err
	}
	for wfRows.Next() {
		var name, graphJSON string
		if err := wfRows.Scan(&name, &graphJSON); err != nil {
			wfRows.Close()
			return err
		}
		var graph workflow.Graph
		if json.Unmarshal([]byte(graphJSON), &graph) == nil {
			for _, node := range graph.Nodes {
				if node.ScriptID == id {
					wfRows.Close()
					return fmt.Errorf("script is used by workflow %q", name)
				}
			}
		}
	}
	wfRows.Close()

	rows, err := db.DB.Query(`SELECT id FROM schedules WHERE script_id=?`, id)
	if err != nil {
		return err
	}
	var scheduleIDs []int
	for rows.Next() {
		var scheduleID int
		if err := rows.Scan(&scheduleID); err != nil {
			rows.Close()
			return err
		}
		scheduleIDs = append(scheduleIDs, scheduleID)
	}
	rows.Close()

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM schedules WHERE script_id=?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`DELETE FROM scripts WHERE id=?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for _, scheduleID := range scheduleIDs {
		scheduler.RemoveJob(scheduleID)
	}
	return nil
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

	recordID, err := script.CreateRecord(scriptID, envSnapshot)
	if err != nil {
		return err
	}

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

	if err := script.StartScript(task, int(recordID), cbs); err != nil {
		script.MarkError(int(recordID))
		return err
	}
	return nil
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
	s.CronExpr = scheduler.NormalizeCron(s.CronExpr)
	if err := scheduler.ValidateCron(s.CronExpr); err != nil {
		return err
	}

	if s.ID == 0 {
		res, err := db.DB.Exec(`INSERT INTO schedules(script_id,cron_expr,enabled,created_at) VALUES(?,?,?,?)`,
			s.ScriptID, s.CronExpr, s.Enabled, time.Now())
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		s.ID = int(id)
		if s.Enabled == 1 {
			if err := a.addScheduleJob(s.ID, s.ScriptID, s.CronExpr); err != nil {
				db.DB.Exec(`DELETE FROM schedules WHERE id=?`, s.ID)
				return err
			}
		}
		return nil
	}

	var oldScriptID, oldEnabled int
	var oldCronExpr string
	if err := db.DB.QueryRow(`SELECT script_id,cron_expr,enabled FROM schedules WHERE id=?`, s.ID).Scan(&oldScriptID, &oldCronExpr, &oldEnabled); err != nil {
		return err
	}
	if _, err := db.DB.Exec(`UPDATE schedules SET script_id=?,cron_expr=?,enabled=? WHERE id=?`,
		s.ScriptID, s.CronExpr, s.Enabled, s.ID); err != nil {
		return err
	}
	scheduler.RemoveJob(s.ID)
	if s.Enabled == 1 {
		if err := a.addScheduleJob(s.ID, s.ScriptID, s.CronExpr); err != nil {
			db.DB.Exec(`UPDATE schedules SET script_id=?,cron_expr=?,enabled=? WHERE id=?`, oldScriptID, oldCronExpr, oldEnabled, s.ID)
			if oldEnabled == 1 {
				_ = a.addScheduleJob(s.ID, oldScriptID, oldCronExpr)
			}
			return err
		}
	}
	return nil
}

func (a *App) addScheduleJob(schedID, scriptID int, cronExpr string) error {
	if scriptID < 0 {
		wfID := -scriptID
		return scheduler.AddJob(schedID, scriptID, cronExpr, func(_ int) { a.RunWorkflow(wfID) })
	}
	return scheduler.AddJob(schedID, scriptID, cronExpr, func(sid int) { a.RunScript(sid, "") })
}

func (a *App) DeleteSchedule(id int) error {
	scheduler.RemoveJob(id)
	_, err := db.DB.Exec(`DELETE FROM schedules WHERE id=?`, id)
	return err
}

func (a *App) ToggleSchedule(id int, enabled bool) error {
	var scriptID, oldEnabled int
	var cronExpr string
	if err := db.DB.QueryRow(`SELECT script_id,cron_expr,enabled FROM schedules WHERE id=?`, id).Scan(&scriptID, &cronExpr, &oldEnabled); err != nil {
		return err
	}
	if (oldEnabled == 1) == enabled {
		return nil
	}
	cronExpr = scheduler.NormalizeCron(cronExpr)
	if enabled {
		if err := scheduler.ValidateCron(cronExpr); err != nil {
			return err
		}
		scheduler.RemoveJob(id)
		if err := a.addScheduleJob(id, scriptID, cronExpr); err != nil {
			return err
		}
	}

	val := 0
	if enabled {
		val = 1
	}
	_, err := db.DB.Exec(`UPDATE schedules SET enabled=?,cron_expr=? WHERE id=?`, val, cronExpr, id)
	if err != nil {
		if enabled {
			scheduler.RemoveJob(id)
		}
		return err
	}

	if !enabled {
		scheduler.RemoveJob(id)
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
	if _, err := workflow.ParseGraph(w.Graph); err != nil {
		return 0, err
	}
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
	_ = a.StopWorkflow(id)
	rows, err := db.DB.Query(`SELECT id FROM schedules WHERE script_id=?`, -id)
	if err != nil {
		return err
	}
	var scheduleIDs []int
	for rows.Next() {
		var scheduleID int
		if err := rows.Scan(&scheduleID); err != nil {
			rows.Close()
			return err
		}
		scheduleIDs = append(scheduleIDs, scheduleID)
	}
	rows.Close()

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM schedules WHERE script_id=?`, -id); err != nil {
		tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`DELETE FROM workflows WHERE id=?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for _, scheduleID := range scheduleIDs {
		scheduler.RemoveJob(scheduleID)
	}
	return nil
}

func (a *App) RunWorkflow(id int) error {
	a.workflowMu.Lock()
	if _, running := a.workflowCancels[id]; running {
		a.workflowMu.Unlock()
		return fmt.Errorf("workflow already running")
	}
	runCtx, cancel := context.WithCancel(a.ctx)
	a.workflowCancels[id] = cancel
	a.workflowMu.Unlock()

	cfg := a.GetGlobalConfig()
	var wfName string
	if err := db.DB.QueryRow(`SELECT name FROM workflows WHERE id=?`, id).Scan(&wfName); err != nil {
		a.workflowMu.Lock()
		delete(a.workflowCancels, id)
		a.workflowMu.Unlock()
		cancel()
		return err
	}
	runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
		"scriptID": -id,
		"status":   "running",
	})
	go func() {
		defer func() {
			a.workflowMu.Lock()
			delete(a.workflowCancels, id)
			a.workflowMu.Unlock()
			cancel()
		}()
		err := workflow.Run(runCtx, id, cfg.EnvFilePath,
			func(runID int, nodeID string, scriptID int, status string) {
				runtime.EventsEmit(a.ctx, "workflow:node-status", map[string]interface{}{
					"workflowId": id, "runId": runID, "nodeId": nodeID, "scriptId": scriptID, "status": status,
				})
				runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
					"scriptID": scriptID, "status": status,
				})
			},
			func(scriptID int, line string, isError bool) {
				entry := map[string]interface{}{
					"scriptID": scriptID, "line": line, "isError": isError,
					"timestamp": time.Now().Format("15:04:05"),
				}
				runtime.EventsEmit(a.ctx, "log:line", entry)
				entry["workflowId"] = id
				runtime.EventsEmit(a.ctx, "workflow:log", entry)
			},
		)
		status := "success"
		if errors.Is(err, context.Canceled) {
			status = "killed"
		} else if err != nil {
			status = "error"
		}
		runtime.EventsEmit(a.ctx, "workflow:status", map[string]interface{}{
			"workflowId": id,
			"status":     status,
		})
		runtime.EventsEmit(a.ctx, "task:status", map[string]interface{}{
			"scriptID": -id,
			"status":   status,
		})
		go notify.Feishu(cfg.LarkCLIPath, cfg.LarkOpenID,
			fmt.Sprintf("[PyLot] 工作流「%s」执行%s", wfName, notify.StatusLabel(status)))
	}()
	return nil
}

func (a *App) StopWorkflow(id int) error {
	a.workflowMu.Lock()
	cancel, running := a.workflowCancels[id]
	a.workflowMu.Unlock()
	if !running {
		return fmt.Errorf("workflow not running")
	}
	cancel()
	return nil
}

func (a *App) GetRunningWorkflows() []int {
	a.workflowMu.Lock()
	defer a.workflowMu.Unlock()
	ids := make([]int, 0, len(a.workflowCancels))
	for id := range a.workflowCancels {
		ids = append(ids, id)
	}
	return ids
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

// ========== Service Management ==========

type ServiceInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	WorkDir   string `json:"work_dir"`
	AutoStart bool   `json:"auto_start"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	URL       string `json:"url"`
	Running   bool   `json:"running"`
	Status    string `json:"status"`
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
	StoppedAt string `json:"stopped_at"`
	ExitCode  int    `json:"exit_code"`
	LastError string `json:"last_error"`
}

type ServicePortStatus struct {
	Configured         bool   `json:"configured"`
	Port               int    `json:"port"`
	Protocol           string `json:"protocol"`
	URL                string `json:"url"`
	Listening          bool   `json:"listening"`
	PID                int    `json:"pid"`
	ParentPID          int    `json:"parent_pid"`
	ProcessName        string `json:"process_name"`
	ProcessPath        string `json:"process_path"`
	ManagedServiceID   int64  `json:"managed_service_id"`
	ManagedServicePID  int    `json:"managed_service_pid"`
	ManagedServiceName string `json:"managed_service_name"`
}

type ServiceLogEntry struct {
	ID        int64  `json:"id"`
	Line      string `json:"line"`
	IsError   bool   `json:"isError"`
	Timestamp string `json:"timestamp"`
}

func (a *App) autoStartServices() {
	rows, err := db.DB.Query(`SELECT id FROM services WHERE auto_start=1`)
	if err != nil {
		return
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	for _, id := range ids {
		a.StartService(id)
	}
}

func (a *App) ListServices() []ServiceInfo {
	rows, err := db.DB.Query(`SELECT id,name,command,work_dir,auto_start,port,protocol FROM services ORDER BY id`)
	if err != nil {
		return []ServiceInfo{}
	}
	defer rows.Close()
	list := []ServiceInfo{}
	for rows.Next() {
		var s ServiceInfo
		var autoStart int
		rows.Scan(&s.ID, &s.Name, &s.Command, &s.WorkDir, &autoStart, &s.Port, &s.Protocol)
		s.AutoStart = autoStart == 1
		s.URL = serviceURL(s.Protocol, s.Port)
		a.applyServiceSnapshot(&s, svc.SnapshotFor(s.ID))
		list = append(list, s)
	}
	return list
}

func (a *App) GetServiceLogs(id int64) []ServiceLogEntry {
	logs := svc.Logs(id)
	out := make([]ServiceLogEntry, 0, len(logs))
	for _, l := range logs {
		out = append(out, ServiceLogEntry{
			ID:        l.ID,
			Line:      l.Line,
			IsError:   l.IsError,
			Timestamp: l.Timestamp,
		})
	}
	return out
}

func (a *App) ClearServiceLogs(id int64) {
	svc.ClearLogs(id)
}

func (a *App) AddService(name, command, workDir string, autoStart bool, port int, protocol string) error {
	protocol, err := validateServiceEndpoint(port, protocol)
	if err != nil {
		return err
	}
	auto := 0
	if autoStart {
		auto = 1
	}
	_, err = db.DB.Exec(`INSERT INTO services(name,command,work_dir,auto_start,port,protocol,created_at) VALUES(?,?,?,?,?,?,?)`,
		name, command, workDir, auto, port, protocol, time.Now())
	return err
}

func (a *App) UpdateService(id int64, name, command, workDir string, autoStart bool, port int, protocol string) error {
	protocol, err := validateServiceEndpoint(port, protocol)
	if err != nil {
		return err
	}
	auto := 0
	if autoStart {
		auto = 1
	}
	_, err = db.DB.Exec(`UPDATE services SET name=?,command=?,work_dir=?,auto_start=?,port=?,protocol=? WHERE id=?`,
		name, command, workDir, auto, port, protocol, id)
	return err
}

func (a *App) SetServiceAutoStart(id int64, autoStart bool) error {
	auto := 0
	if autoStart {
		auto = 1
	}
	_, err := db.DB.Exec(`UPDATE services SET auto_start=? WHERE id=?`, auto, id)
	return err
}

func (a *App) DeleteService(id int64) error {
	snap, _ := svc.Stop(id)
	a.emitServiceStatus(snap)
	svc.Forget(id)
	_, err := db.DB.Exec(`DELETE FROM services WHERE id=?`, id)
	return err
}

func (a *App) StartService(id int64) error {
	row := db.DB.QueryRow(`SELECT command,work_dir,port FROM services WHERE id=?`, id)
	var command, workDir string
	var port int
	if err := row.Scan(&command, &workDir, &port); err != nil {
		return err
	}
	if port > 0 {
		owner, err := svc.FindPortOwner(port)
		if err != nil {
			return err
		}
		if owner != nil {
			return fmt.Errorf("端口 %d 已被 PID %d 占用", port, owner.PID)
		}
	}
	err := svc.Start(id, command, workDir, svc.Callbacks{
		OnLog: func(entry svc.LogEntry) {
			runtime.EventsEmit(a.ctx, "service:log", map[string]any{
				"id":        entry.ID,
				"line":      entry.Line,
				"isError":   entry.IsError,
				"timestamp": entry.Timestamp,
			})
		},
		OnStatus: func(snap svc.Snapshot) {
			a.emitServiceStatus(snap)
		},
	})
	return err
}

func (a *App) StopService(id int64) error {
	snap, err := svc.Stop(id)
	a.emitServiceStatus(snap)
	return err
}

func (a *App) RestartService(id int64) error {
	wasRunning := svc.IsRunning(id)
	snap, err := svc.Stop(id)
	a.emitServiceStatus(snap)
	if err != nil {
		return err
	}
	if wasRunning {
		if err := svc.WaitStopped(id, 5*time.Second); err != nil {
			return err
		}
		var port int
		if err := db.DB.QueryRow(`SELECT port FROM services WHERE id=?`, id).Scan(&port); err != nil {
			return err
		}
		if port > 0 {
			if err := svc.WaitPortFree(port, 5*time.Second); err != nil {
				return err
			}
		}
	}
	return a.StartService(id)
}

func (a *App) GetServicePortStatus(id int64) (ServicePortStatus, error) {
	var port int
	var protocol string
	if err := db.DB.QueryRow(`SELECT port,protocol FROM services WHERE id=?`, id).Scan(&port, &protocol); err != nil {
		return ServicePortStatus{}, err
	}
	status := ServicePortStatus{
		Configured: port > 0,
		Port:       port,
		Protocol:   protocol,
		URL:        serviceURL(protocol, port),
	}
	if port == 0 {
		return status, nil
	}
	owner, err := svc.FindPortOwner(port)
	if err != nil {
		return ServicePortStatus{}, err
	}
	if owner == nil {
		return status, nil
	}
	status.Listening = true
	status.PID = owner.PID
	status.ParentPID = owner.ParentPID
	status.ProcessName = owner.ProcessName
	status.ProcessPath = owner.ProcessPath
	services := a.ListServices()
	rootPIDs := make([]int, 0, len(services))
	for _, item := range services {
		if item.PID > 0 {
			rootPIDs = append(rootPIDs, item.PID)
		}
	}
	rootPID := svc.FindProcessTreeRoot(owner.PID, rootPIDs)
	for _, item := range services {
		if item.PID == rootPID {
			status.ManagedServiceID = item.ID
			status.ManagedServicePID = item.PID
			status.ManagedServiceName = item.Name
			break
		}
	}
	return status, nil
}

func (a *App) TerminatePortOwnerAndStartService(id int64, expectedPID int) error {
	var port int
	if err := db.DB.QueryRow(`SELECT port FROM services WHERE id=?`, id).Scan(&port); err != nil {
		return err
	}
	if port == 0 {
		return fmt.Errorf("服务未配置端口")
	}
	if err := svc.KillPortOwner(port, expectedPID); err != nil {
		return err
	}
	return a.StartService(id)
}

func validateServiceEndpoint(port int, protocol string) (string, error) {
	if port < 0 || port > 65535 {
		return "", fmt.Errorf("端口必须在 1-65535 之间，留空表示不管理端口")
	}
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol == "" {
		protocol = "http"
	}
	if protocol != "http" && protocol != "https" {
		return "", fmt.Errorf("协议只支持 http 或 https")
	}
	return protocol, nil
}

func serviceURL(protocol string, port int) string {
	if port == 0 {
		return ""
	}
	if protocol == "" {
		protocol = "http"
	}
	return fmt.Sprintf("%s://127.0.0.1:%d", protocol, port)
}

func (a *App) applyServiceSnapshot(s *ServiceInfo, snap svc.Snapshot) {
	s.Running = snap.Running
	s.Status = snap.Status
	s.PID = snap.PID
	s.StartedAt = snap.StartedAt
	s.StoppedAt = snap.StoppedAt
	s.ExitCode = snap.ExitCode
	s.LastError = snap.LastError
}

func (a *App) emitServiceStatus(snap svc.Snapshot) {
	runtime.EventsEmit(a.ctx, "service:status", map[string]any{
		"id":         snap.ID,
		"running":    snap.Running,
		"status":     snap.Status,
		"pid":        snap.PID,
		"started_at": snap.StartedAt,
		"stopped_at": snap.StoppedAt,
		"exit_code":  snap.ExitCode,
		"last_error": snap.LastError,
	})
}
