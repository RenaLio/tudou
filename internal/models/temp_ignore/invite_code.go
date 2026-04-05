package temp_ignore

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// InviteCodeStatus 邀请码状态
type InviteCodeStatus string

const (
	InviteCodeStatusActive   InviteCodeStatus = "active"   // 有效
	InviteCodeStatusExpired  InviteCodeStatus = "expired"  // 已过期
	InviteCodeStatusDepleted InviteCodeStatus = "depleted" // 已用完
	InviteCodeStatusDisabled InviteCodeStatus = "disabled" // 已禁用
)

// InviteCode 邀请码模型
type InviteCode struct {
	ID        int64                 `gorm:"primaryKey;column:id" json:"id"`
	Code      string                `gorm:"column:code;type:varchar(64);not null;uniqueIndex:idx_invite_code" json:"code"`
	CreatedBy int64                 `gorm:"column:created_by;type:bigint;not null;index:idx_invite_creator" json:"created_by"`
	MaxUses   int                   `gorm:"column:max_uses;type:int;default:1" json:"max_uses"`     // 最大使用次数，-1表示无限
	UsedCount int                   `gorm:"column:used_count;type:int;default:0" json:"used_count"` // 已使用次数
	ExpiresAt *time.Time            `gorm:"column:expires_at;type:timestamp" json:"expires_at,omitempty"`
	Status    InviteCodeStatus      `gorm:"column:status;type:varchar(32);default:'active';index:idx_invite_status" json:"status"`
	Note      string                `gorm:"column:note;type:varchar(256)" json:"note"` // 备注
	CreatedAt time.Time             `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_invite_deleted_at" json:"-"`

	// 关联关系
	Creator models.User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName 指定表名
func (*InviteCode) TableName() string {
	return "invite_codes"
}

// BeforeCreate 创建前钩子，生成雪花ID
func (i *InviteCode) BeforeCreate(tx *gorm.DB) error {
	if i.ID == 0 {
		node, err := snowflake.NewNode(1)
		if err != nil {
			return err
		}
		i.ID = node.Generate().Int64()
	}
	return nil
}

// IsAvailable 检查邀请码是否可用
func (i *InviteCode) IsAvailable() bool {
	if i.Status != InviteCodeStatusActive {
		return false
	}
	if i.ExpiresAt != nil && i.ExpiresAt.Before(time.Now()) {
		return false
	}
	if i.MaxUses >= 0 && i.UsedCount >= i.MaxUses {
		return false
	}
	return true
}

// IsExpired 检查是否过期
func (i *InviteCode) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return i.ExpiresAt.Before(time.Now())
}

// IsDepleted 检查是否已用完
func (i *InviteCode) IsDepleted() bool {
	if i.MaxUses < 0 {
		return false
	}
	return i.UsedCount >= i.MaxUses
}

// Use 使用邀请码
func (i *InviteCode) Use() bool {
	if !i.IsAvailable() {
		return false
	}
	i.UsedCount++
	if i.IsDepleted() {
		i.Status = InviteCodeStatusDepleted
	}
	return true
}

// UpdateStatus 更新状态
func (i *InviteCode) UpdateStatus() {
	if i.Status == InviteCodeStatusDisabled {
		return
	}
	if i.IsExpired() {
		i.Status = InviteCodeStatusExpired
	} else if i.IsDepleted() {
		i.Status = InviteCodeStatusDepleted
	} else {
		i.Status = InviteCodeStatusActive
	}
}

// RemainingUses 剩余使用次数
func (i *InviteCode) RemainingUses() int {
	if i.MaxUses < 0 {
		return -1 // 无限
	}
	remaining := i.MaxUses - i.UsedCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// InviteCodeQuery 邀请码查询条件
type InviteCodeQuery struct {
	CreatedBy int64
	Status    InviteCodeStatus
	Code      string
}
