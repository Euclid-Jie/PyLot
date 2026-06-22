package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const (
	StatusStopped  = "stopped"
	StatusStarting = "starting"
	StatusRunning  = "running"
	StatusStopping = "stopping"
	StatusExited   = "exited"
	StatusFailed   = "failed"

	maxLogLines = 1000
)

type LogEntry struct {
	ID        int64  `json:"id"`
	Line      string `json:"line"`
	IsError   bool   `json:"isError"`
	Timestamp string `json:"timestamp"`
}

type Snapshot struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
	StoppedAt string `json:"stopped_at"`
	ExitCode  int    `json:"exit_code"`
	LastError string `json:"last_error"`
}

type Callbacks struct {
	OnLog    func(LogEntry)
	OnStatus func(Snapshot)
}

type lineWriter struct {
	buf    bytes.Buffer
	onLine func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		b := w.buf.Bytes()
		idx := bytes.IndexByte(b, '\n')
		if idx < 0 {
			break
		}
		line := string(b[:idx])
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		w.onLine(line)
		w.buf.Next(idx + 1)
	}
	return len(p), nil
}

func (w *lineWriter) Flush() {
	if w.buf.Len() == 0 {
		return
	}
	line := strings.TrimSuffix(w.buf.String(), "\r")
	w.buf.Reset()
	if line != "" {
		w.onLine(line)
	}
}

type runtimeState struct {
	cmd           *exec.Cmd
	status        string
	pid           int
	startedAt     time.Time
	stoppedAt     time.Time
	exitCode      int
	lastError     string
	stopRequested bool
	logs          []LogEntry
}

var (
	mu     sync.Mutex
	states = map[int64]*runtimeState{}
)

func Start(id int64, command, workDir string, cbs Callbacks) error {
	mu.Lock()
	st := ensureStateLocked(id)
	if st.cmd != nil {
		snap := snapshotLocked(id, st)
		mu.Unlock()
		emitStatus(cbs, snap)
		return fmt.Errorf("service is already running")
	}
	st.status = StatusStarting
	st.pid = 0
	st.startedAt = time.Time{}
	st.stoppedAt = time.Time{}
	st.exitCode = 0
	st.lastError = ""
	st.stopRequested = false
	st.logs = nil
	snap := snapshotLocked(id, st)
	mu.Unlock()
	emitStatus(cbs, snap)

	exe, args, err := parseCommand(command)
	if err != nil {
		snap = failStart(id, err)
		emitStatus(cbs, snap)
		return err
	}

	exe = resolveExecutable(exe, workDir)
	cmd := exec.Command(exe, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var stdoutWriter, stderrWriter *lineWriter
	stdoutWriter = &lineWriter{onLine: func(line string) { appendLog(id, line, false, cbs) }}
	stderrWriter = &lineWriter{onLine: func(line string) { appendLog(id, line, true, cbs) }}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := cmd.Start(); err != nil {
		snap = failStart(id, err)
		emitStatus(cbs, snap)
		return err
	}

	mu.Lock()
	st = ensureStateLocked(id)
	st.cmd = cmd
	st.status = StatusRunning
	st.pid = cmd.Process.Pid
	st.startedAt = time.Now()
	snap = snapshotLocked(id, st)
	mu.Unlock()
	emitStatus(cbs, snap)

	go waitForExit(id, cmd, stdoutWriter, stderrWriter, cbs)
	return nil
}

func Stop(id int64) (Snapshot, error) {
	mu.Lock()
	st := ensureStateLocked(id)
	if st.cmd == nil {
		st.status = StatusStopped
		st.pid = 0
		st.stopRequested = false
		snap := snapshotLocked(id, st)
		mu.Unlock()
		return snap, nil
	}

	cmd := st.cmd
	st.status = StatusStopping
	st.stopRequested = true
	snap := snapshotLocked(id, st)
	mu.Unlock()

	err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid)).Run()
	return snap, err
}

func SnapshotFor(id int64) Snapshot {
	mu.Lock()
	defer mu.Unlock()
	return snapshotLocked(id, ensureStateLocked(id))
}

func Logs(id int64) []LogEntry {
	mu.Lock()
	defer mu.Unlock()
	st := ensureStateLocked(id)
	out := make([]LogEntry, len(st.logs))
	copy(out, st.logs)
	return out
}

func ClearLogs(id int64) {
	mu.Lock()
	ensureStateLocked(id).logs = nil
	mu.Unlock()
}

func IsRunning(id int64) bool {
	mu.Lock()
	defer mu.Unlock()
	return ensureStateLocked(id).cmd != nil
}

func Forget(id int64) {
	mu.Lock()
	delete(states, id)
	mu.Unlock()
}

func StopAll() {
	mu.Lock()
	ids := make([]int64, 0, len(states))
	for id, st := range states {
		if st.cmd != nil {
			ids = append(ids, id)
		}
	}
	mu.Unlock()
	for _, id := range ids {
		_, _ = Stop(id)
	}
}

func parseCommand(command string) (string, []string, error) {
	parts, err := windows.DecomposeCommandLine(strings.TrimSpace(command))
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", nil, fmt.Errorf("empty command")
	}
	return parts[0], parts[1:], nil
}

func resolveExecutable(exe, workDir string) string {
	if exe == "" || filepath.IsAbs(exe) || workDir == "" {
		return exe
	}
	candidate := filepath.Join(workDir, exe)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return exe
}

func waitForExit(id int64, cmd *exec.Cmd, stdoutWriter, stderrWriter *lineWriter, cbs Callbacks) {
	err := cmd.Wait()
	stdoutWriter.Flush()
	stderrWriter.Flush()

	mu.Lock()
	st, ok := states[id]
	if !ok || st.cmd != cmd {
		mu.Unlock()
		return
	}

	st.cmd = nil
	st.pid = 0
	st.stoppedAt = time.Now()
	st.exitCode = processExitCode(cmd, err)
	switch {
	case st.stopRequested:
		st.status = StatusStopped
		st.lastError = ""
	case err != nil:
		st.status = StatusFailed
		st.lastError = err.Error()
	default:
		st.status = StatusExited
		st.lastError = ""
	}
	st.stopRequested = false
	snap := snapshotLocked(id, st)
	mu.Unlock()

	emitStatus(cbs, snap)
}

func failStart(id int64, err error) Snapshot {
	mu.Lock()
	defer mu.Unlock()
	st := ensureStateLocked(id)
	st.cmd = nil
	st.status = StatusFailed
	st.pid = 0
	st.stoppedAt = time.Now()
	st.exitCode = -1
	st.lastError = err.Error()
	st.stopRequested = false
	return snapshotLocked(id, st)
}

func appendLog(id int64, line string, isError bool, cbs Callbacks) {
	entry := LogEntry{
		ID:        id,
		Line:      line,
		IsError:   isError,
		Timestamp: time.Now().Format("15:04:05"),
	}

	mu.Lock()
	st := ensureStateLocked(id)
	st.logs = append(st.logs, entry)
	if len(st.logs) > maxLogLines {
		st.logs = st.logs[len(st.logs)-maxLogLines:]
	}
	mu.Unlock()

	if cbs.OnLog != nil {
		cbs.OnLog(entry)
	}
}

func ensureStateLocked(id int64) *runtimeState {
	st, ok := states[id]
	if !ok {
		st = &runtimeState{status: StatusStopped}
		states[id] = st
	}
	return st
}

func snapshotLocked(id int64, st *runtimeState) Snapshot {
	return Snapshot{
		ID:        id,
		Status:    st.status,
		Running:   st.cmd != nil,
		PID:       st.pid,
		StartedAt: formatTime(st.startedAt),
		StoppedAt: formatTime(st.stoppedAt),
		ExitCode:  st.exitCode,
		LastError: st.lastError,
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func processExitCode(cmd *exec.Cmd, err error) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	if err != nil {
		return -1
	}
	return 0
}

func emitStatus(cbs Callbacks, snap Snapshot) {
	if cbs.OnStatus != nil {
		cbs.OnStatus(snap)
	}
}
