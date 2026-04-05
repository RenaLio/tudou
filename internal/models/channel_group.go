package models

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

// ChannelGroup 渠道组模型
type ChannelGroup struct {
	ID                  int64                 `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	Name                string                `gorm:"column:name;type:varchar(128);not null;uniqueIndex:idx_group_name" json:"name"`
	NameRemark          string                `gorm:"column:name_remark;type:varchar(256)" json:"nameRemark"`
	Description         string                `gorm:"column:description;type:text" json:"description"`
	PermissionNum       int32                 `gorm:"column:permission_num;type:int;default:0" json:"permissionNum"`
	LoadBalanceStrategy LoadBalanceStrategy   `gorm:"column:load_balance_strategy;type:varchar(32);default:'performance'" json:"loadBalanceStrategy"`
	CreatedAt           time.Time             `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt           soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_group_deleted_at" json:"-"`

	// 关联关系
	Channels []Channel `gorm:"many2many:group_channels;" json:"channels,omitempty"`
	Tokens   []Token   `gorm:"foreignKey:GroupID" json:"tokens,omitempty"`
}

// TableName 指定表名
func (*ChannelGroup) TableName() string {
	return "channel_groups"
}

// GroupChannel 渠道组与渠道的关联表
type GroupChannel struct {
	GroupID   int64     `gorm:"column:group_id;primaryKey;index:idx_group_channel_group"`
	ChannelID int64     `gorm:"column:channel_id;primaryKey;index:idx_group_channel_channel"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName 指定表名
func (GroupChannel) TableName() string {
	return "group_channels"
}
