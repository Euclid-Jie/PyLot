package db

import "time"

const (
	TriggerSourceUnknown  = "unknown"
	TriggerSourceManual   = "manual"
	TriggerSourceSchedule = "schedule"
	TriggerSourceWorkflow = "workflow"
)

type GlobalConfig struct {
	ID          int       `json:"id"`
	EnvFilePath string    `json:"envFilePath"`
	LarkCLIPath string    `json:"larkCliPath"`
	LarkOpenID  string    `json:"larkOpenId"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Script struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	ListID          int       `json:"listId"`
	InterpreterPath string    `json:"interpreterPath"`
	WorkDir         string    `json:"workDir"`
	ScriptPath      string    `json:"scriptPath"`
	LaunchMode      string    `json:"launchMode"`
	FixedArgs       string    `json:"fixedArgs"`
	PrivateEnv      string    `json:"privateEnv"`
	TimeoutSeconds  int       `json:"timeoutSeconds"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ScriptList struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	SortOrder   int    `json:"sortOrder"`
	ScriptCount int    `json:"scriptCount"`
}

type Schedule struct {
	ID        int       `json:"id"`
	ScriptID  int       `json:"scriptId"`
	CronExpr  string    `json:"cronExpr"`
	Enabled   int       `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
}

type RunRecord struct {
	ID            int        `json:"id"`
	ScriptID      int        `json:"scriptId"`
	StartedAt     time.Time  `json:"startedAt"`
	EndedAt       *time.Time `json:"endedAt"`
	Status        string     `json:"status"`
	LogOutput     string     `json:"logOutput"`
	IsError       int        `json:"isError"`
	EnvSnapshot   string     `json:"envSnapshot"`
	TriggerSource string     `json:"triggerSource"`
	ScheduleID    int        `json:"scheduleId"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type RecentRun struct {
	RecordID      int        `json:"recordId"`
	TargetID      int        `json:"targetId"`
	TargetType    string     `json:"targetType"`
	TargetName    string     `json:"targetName"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"startedAt"`
	EndedAt       *time.Time `json:"endedAt"`
	IsError       int        `json:"isError"`
	TriggerSource string     `json:"triggerSource"`
	ScheduleID    int        `json:"scheduleId"`
}

type RunningTask struct {
	ScriptID  int       `json:"scriptId"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
}

type Workflow struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Graph     string    `json:"graph"` // JSON: {nodes:[{id,scriptId,x,y}], edges:[{source,target}]}
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type WorkflowRun struct {
	ID            int        `json:"id"`
	WorkflowID    int        `json:"workflowId"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"startedAt"`
	EndedAt       *time.Time `json:"endedAt"`
	TriggerSource string     `json:"triggerSource"`
	ScheduleID    int        `json:"scheduleId"`
}

type WorkflowRunNode struct {
	ID            int        `json:"id"`
	WorkflowRunID int        `json:"workflowRunId"`
	NodeID        string     `json:"nodeId"`
	ScriptID      int        `json:"scriptId"`
	ScriptName    string     `json:"scriptName"`
	Status        string     `json:"status"`
	StartedAt     *time.Time `json:"startedAt"`
	EndedAt       *time.Time `json:"endedAt"`
	RunRecordID   int        `json:"runRecordId"`
	SortOrder     int        `json:"sortOrder"`
}
