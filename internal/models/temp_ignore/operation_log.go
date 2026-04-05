package temp_ignore

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

// OperationType 操作类型
type OperationType string

const (
	OperationTypeCreate OperationType = "create" // 创建
	OperationTypeUpdate OperationType = "update" // 更新
	OperationTypeDelete OperationType = "delete" // 删除
	OperationTypeQuery  OperationType = "query"  // 查询
	OperationTypeLogin  OperationType = "login"  // 登录
	OperationTypeLogout OperationType = "logout" // 登出
	OperationTypeOther  OperationType = "other"  // 其他
)

// OperationTarget 操作对象类型
type OperationTarget string

const (
	OperationTargetUser         OperationTarget = "user"          // 用户
	OperationTargetToken        OperationTarget = "token"         // 令牌
	OperationTargetChannel      OperationTarget = "channel"       // 渠道
	OperationTargetChannelGroup OperationTarget = "channel_group" // 渠道组
	OperationTargetModel        OperationTarget = "model"         // 模型
	OperationTargetConfig       OperationTarget = "config"        // 配置
	OperationTargetSystem       OperationTarget = "system"        // 系统
)

// OperationStatus 操作状态
type OperationStatus string

const (
	OperationStatusSuccess OperationStatus = "success" // 成功
	OperationStatusFail    OperationStatus = "fail"    // 失败
)

// OperationDetail 操作详情
type OperationDetail struct {
	Before interface{} `json:"before,omitempty"` // 操作前数据
	After  interface{} `json:"after,omitempty"`  // 操作后数据
	Extra  interface{} `json:"extra,omitempty"`  // 额外信息
}

// Value 实现 driver.Valuer 接口
func (d *OperationDetail) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Scan 实现 sql.Scanner 接口
func (d *OperationDetail) Scan(value interface{}) error {
	if value == nil {
		*d = OperationDetail{}
		return nil
	}
	return models.unmarshalJSONValue(value, d)
}

// OperationLog 操作日志模型
type OperationLog struct {
	ID          int64           `gorm:"primaryKey;column:id" json:"id"`
	UserID      int64           `gorm:"column:user_id;type:bigint;not null;index:idx_oplog_user" json:"user_id"`
	Type        OperationType   `gorm:"column:type;type:varchar(32);not null;index:idx_oplog_type" json:"type"`
	Target      OperationTarget `gorm:"column:target;type:varchar(32);not null;index:idx_oplog_target" json:"target"`
	TargetID    string          `gorm:"column:target_id;type:varchar(64);index:idx_oplog_target_id" json:"target_id"`
	Action      string          `gorm:"column:action;type:varchar(128);not null" json:"action"` // 具体动作，如：create_token
	Description string          `gorm:"column:description;type:text" json:"description"`
	Detail      OperationDetail `gorm:"column:detail;type:json" json:"detail"`
	Status      OperationStatus `gorm:"column:status;type:varchar(32);not null;index:idx_oplog_status" json:"status"`
	ErrorMsg    string          `gorm:"column:error_msg;type:text" json:"error_msg,omitempty"`
	IP          string          `gorm:"column:ip;type:varchar(64)" json:"ip"`
	UserAgent   string          `gorm:"column:user_agent;type:varchar(512)" json:"user_agent"`
	Duration    int64           `gorm:"column:duration;type:bigint;default:0" json:"duration"` // 操作耗时(ms)
	CreatedAt   time.Time       `gorm:"column:created_at;type:timestamp;not null;index:idx_oplog_created" json:"created_at"`
}

// TableName 指定表名
func (*OperationLog) TableName() string {
	return "operation_logs"
}

// BeforeCreate 创建前钩子，生成雪花ID
func (l *OperationLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == 0 {
		node, err := snowflake.NewNode(1)
		if err != nil {
			return err
		}
		l.ID = node.Generate().Int64()
	}
	return nil
}

// IsSuccess 是否成功
func (l *OperationLog) IsSuccess() bool {
	return l.Status == OperationStatusSuccess
}

// SetSuccess 设置成功
func (l *OperationLog) SetSuccess() {
	l.Status = OperationStatusSuccess
}

// SetFail 设置失败
func (l *OperationLog) SetFail(errMsg string) {
	l.Status = OperationStatusFail
	l.ErrorMsg = errMsg
}

// SetDetail 设置详情
func (l *OperationLog) SetDetail(before, after, extra interface{}) {
	l.Detail = OperationDetail{
		Before: before,
		After:  after,
		Extra:  extra,
	}
}

// OperationLogQuery 操作日志查询条件
type OperationLogQuery struct {
	UserID    int64
	Type      OperationType
	Target    OperationTarget
	TargetID  string
	Status    OperationStatus
	StartTime *time.Time
	EndTime   *time.Time
}

// NewOperationLog 创建操作日志（便捷方法）
func NewOperationLog(userID int64, opType OperationType, target OperationTarget, targetID, action string) *OperationLog {
	return &OperationLog{
		UserID:   userID,
		Type:     opType,
		Target:   target,
		TargetID: targetID,
		Action:   action,
		Status:   OperationStatusSuccess,
	}
}
