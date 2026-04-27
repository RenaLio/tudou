package models

import "time"

type AggregationTaskStatus int8

const (
	AggregationTaskStatusPending AggregationTaskStatus = 0
	AggregationTaskStatusRunning AggregationTaskStatus = 1
	AggregationTaskStatusDone    AggregationTaskStatus = 2
	AggregationTaskStatusFailed  AggregationTaskStatus = 3
)

type AggregationTask struct {
	ID         int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TaskName   string     `gorm:"column:task_name;type:varchar(128);not null;index" json:"taskName"`
	StartID    int64      `gorm:"column:start_id;type:bigint;default:0" json:"startID,string"`
	EndID      int64      `gorm:"column:end_id;type:bigint;default:0" json:"endID,string"`
	Status     int8       `gorm:"column:status;type:int;default:0;comment:任务状态，0:待处理, 1:处理中, 2:已完成, 3:失败" json:"status"`
	RetryCount int32      `gorm:"column:retry_count;type:int;default:0;comment:重试次数" json:"retryCount"`
	ErrorMsg   string     `gorm:"column:error_msg;type:text" json:"errorMsg"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	FinishedAt *time.Time `gorm:"column:finished_at;" json:"finishedAt"`
}

func (*AggregationTask) TableName() string {
	return "aggregation_tasks"
}
