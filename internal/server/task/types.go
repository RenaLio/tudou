package task

import "time"

type taskEntry struct {
	task   Task
	config TaskConfig
	state  TaskState
}

type TaskState struct {
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	Interval       time.Duration `json:"interval"`
	Timeout        time.Duration `json:"timeout"`
	AllowOverlap   bool          `json:"allowOverlap"`
	Running        bool          `json:"running"`
	ActiveRuns     int           `json:"activeRuns"`
	LastStartedAt  *time.Time    `json:"lastStartedAt,omitempty"`
	LastFinishedAt *time.Time    `json:"lastFinishedAt,omitempty"`
	LastDuration   time.Duration `json:"lastDuration"`
	NextRunAt      *time.Time    `json:"nextRunAt,omitempty"`
	LastError      string        `json:"lastError,omitempty"`
	RunCount       uint64        `json:"runCount"`
	SuccessCount   uint64        `json:"successCount"`
	FailureCount   uint64        `json:"failureCount"`
}

type TaskConfig struct {
	Enabled      bool          `json:"enabled"`
	Interval     time.Duration `json:"interval"`
	Timeout      time.Duration `json:"timeout"`
	AllowOverlap bool          `json:"allowOverlap"`
}

type taskRunRequest struct {
	name      string
	task      Task
	timeout   time.Duration
	startedAt time.Time
}
