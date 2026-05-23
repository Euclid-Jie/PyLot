package scheduler

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type ScheduleInfo struct {
	ScheduleID int       `json:"scheduleId"`
	ScriptID   int       `json:"scriptId"`
	ScriptName string    `json:"scriptName"`
	CronExpr   string    `json:"cronExpr"`
	NextRun    time.Time `json:"nextRun"`
	Enabled    bool      `json:"enabled"`
}

type jobEntry struct {
	entryID  cron.EntryID
	scriptID int
	cronExpr string
}

var (
	c       *cron.Cron
	jobs    = map[int]jobEntry{} // scheduleID -> jobEntry
	mu      sync.Mutex
)

func Init() {
	c = cron.New()
}

func Start() { c.Start() }
func Stop()  { c.Stop() }

func AddJob(scheduleID, scriptID int, cronExpr string, runFunc func(int)) error {
	mu.Lock()
	defer mu.Unlock()
	id, err := c.AddFunc(cronExpr, func() { runFunc(scriptID) })
	if err != nil {
		return err
	}
	jobs[scheduleID] = jobEntry{entryID: id, scriptID: scriptID, cronExpr: cronExpr}
	return nil
}

func RemoveJob(scheduleID int) {
	mu.Lock()
	defer mu.Unlock()
	if e, ok := jobs[scheduleID]; ok {
		c.Remove(e.entryID)
		delete(jobs, scheduleID)
	}
}

func GetNextRunTimes() []ScheduleInfo {
	mu.Lock()
	defer mu.Unlock()
	result := make([]ScheduleInfo, 0, len(jobs))
	for schedID, job := range jobs {
		entry := c.Entry(job.entryID)
		result = append(result, ScheduleInfo{
			ScheduleID: schedID,
			ScriptID:   job.scriptID,
			CronExpr:   job.cronExpr,
			NextRun:    entry.Next,
			Enabled:    true,
		})
	}
	return result
}
