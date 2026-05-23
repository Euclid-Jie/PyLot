package db

import "time"

type GlobalConfig struct {
	ID          int       `json:"id"`
	EnvFilePath string    `json:"envFilePath"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Script struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
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

type Schedule struct {
	ID        int       `json:"id"`
	ScriptID  int       `json:"scriptId"`
	CronExpr  string    `json:"cronExpr"`
	Enabled   int       `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
}

type RunRecord struct {
	ID          int        `json:"id"`
	ScriptID    int        `json:"scriptId"`
	StartedAt   time.Time  `json:"startedAt"`
	EndedAt     *time.Time `json:"endedAt"`
	Status      string     `json:"status"`
	LogOutput   string     `json:"logOutput"`
	IsError     int        `json:"isError"`
	EnvSnapshot string     `json:"envSnapshot"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type RunningTask struct {
	ScriptID  int       `json:"scriptId"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
}
