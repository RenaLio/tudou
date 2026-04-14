package models

import (
	"database/sql/driver"

	"time"

	"github.com/goccy/go-json"
	"gorm.io/plugin/soft_delete"
)

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusEnabled  UserStatus = "enabled"  // 启用
	UserStatusDisabled UserStatus = "disabled" // 禁用
	UserStatusLocked   UserStatus = "locked"   // 锁定
)

// UserRole 用户角色
type UserRole string

const (
	UserRoleAdmin UserRole = "admin" // 管理员
	UserRoleUser  UserRole = "user"  // 普通用户
	UserRoleGuest UserRole = "guest" // 访客
)

// UserSettings 用户配置
type UserSettings struct {
	Theme       string `json:"theme,omitempty"`       // 主题
	Language    string `json:"language,omitempty"`    // 语言
	Timezone    string `json:"timezone,omitempty"`    // 时区
	NotifyEmail bool   `json:"notifyEmail,omitempty"` // 邮件通知
	NotifySMS   bool   `json:"notifySMS,omitempty"`   // 短信通知
}

// Value 实现 driver.Valuer 接口
func (s UserSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *UserSettings) Scan(value interface{}) error {
	if value == nil {
		*s = UserSettings{}
		return nil
	}
	return unmarshalJSONValue(value, s)
}

// User 用户模型
type User struct {
	ID          int64                 `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	Username    string                `gorm:"column:username;type:varchar(64);not null;uniqueIndex:idx_user_username" json:"username"`
	Password    string                `gorm:"column:password;type:varchar(256);not null" json:"-"`
	Email       string                `gorm:"column:email;type:varchar(128);uniqueIndex:idx_user_email" json:"email"`
	Phone       string                `gorm:"column:phone;type:varchar(32);uniqueIndex:idx_user_phone" json:"phone"`
	Nickname    string                `gorm:"column:nickname;type:varchar(128)" json:"nickname"`
	Avatar      string                `gorm:"column:avatar;type:varchar(512)" json:"avatar"`
	Status      UserStatus            `gorm:"column:status;type:varchar(32);default:'enabled';index:idx_user_status" json:"status"`
	Role        UserRole              `gorm:"column:role;type:varchar(32);default:'user'" json:"role"`
	LastLoginAt *time.Time            `gorm:"column:last_login_at;type:timestamp" json:"lastLoginAt,omitempty"`
	LastLoginIP string                `gorm:"column:last_login_ip;type:varchar(64)" json:"lastLoginIP,omitempty"`
	LoginCount  int64                 `gorm:"column:login_count;type:bigint;default:0" json:"loginCount"`
	Settings    UserSettings          `gorm:"column:settings;type:json" json:"settings"`
	CreatedAt   time.Time             `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt   soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_user_deleted_at;uniqueIndex:idx_user_username" json:"-"`

	// 关联关系
	Tokens []Token   `gorm:"foreignKey:UserID" json:"tokens,omitempty"`
	Stats  UserStats `gorm:"foreignKey:UserID" json:"stats,omitempty"`
}

// TableName 指定表名
func (*User) TableName() string {
	return "users"
}

// IsAvailable 检查用户是否可用
func (u *User) IsAvailable() bool {
	return u.Status == UserStatusEnabled
}

// IsAdmin 检查是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// RecordLogin 记录登录
func (u *User) RecordLogin(ip string) {
	u.LoginCount++
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
}

// UpdatePassword 更新密码
func (u *User) UpdatePassword(newPassword string) {
	u.Password = newPassword
}

type UserStats struct {
	UserID                    int64 `gorm:"column:user_id;primaryKey;index:idx_user_stats_user" json:"userID,string"`
	InputToken                int64 `gorm:"column:input_token;type:bigint;default:0;comment:输入token数" json:"inputToken"`
	OutputToken               int64 `gorm:"column:output_token;type:bigint;default:0;comment:输出token数" json:"outputToken"`
	CachedCreationInputTokens int64 `gorm:"column:cached_creation_input_tokens;type:bigint;default:0;comment:缓存创建输入token数" json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64 `gorm:"column:cached_read_input_tokens;type:bigint;default:0;comment:缓存读取输入token数" json:"cachedReadInputTokens"`
	RequestSuccess            int64 `gorm:"column:request_success;type:bigint;default:0;comment:请求成功数" json:"requestSuccess"`
	RequestFailed             int64 `gorm:"column:request_failed;type:bigint;default:0;comment:请求失败数" json:"requestFailed"`
	TotalCostMicros           int64 `gorm:"column:total_cost_micros;type:bigint;default:0;comment:总成本，单位 micros" json:"totalCostMicros"`
}
