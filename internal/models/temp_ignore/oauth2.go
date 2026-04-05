package temp_ignore

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// OAuth2ProviderType 第三方OAuth2提供商类型
type OAuth2ProviderType string

const (
	OAuth2ProviderGitHub   OAuth2ProviderType = "github"   // GitHub
	OAuth2ProviderGoogle   OAuth2ProviderType = "google"   // Google
	OAuth2ProviderWeChat   OAuth2ProviderType = "wechat"   // 微信
	OAuth2ProviderWeibo    OAuth2ProviderType = "weibo"    // 微博
	OAuth2ProviderLark     OAuth2ProviderType = "lark"     // 飞书
	OAuth2ProviderDingTalk OAuth2ProviderType = "dingtalk" // 钉钉
)

// OAuth2ProviderStatus 提供商状态
type OAuth2ProviderStatus string

const (
	OAuth2ProviderStatusEnabled  OAuth2ProviderStatus = "enabled"  // 启用
	OAuth2ProviderStatusDisabled OAuth2ProviderStatus = "disabled" // 禁用
)

// OAuth2Provider 第三方OAuth2提供商配置
type OAuth2Provider struct {
	ID           int64                 `gorm:"primaryKey;column:id" json:"id"`
	Type         OAuth2ProviderType    `gorm:"column:type;type:varchar(32);not null;uniqueIndex:idx_oauth2_provider_type" json:"type"`
	Name         string                `gorm:"column:name;type:varchar(64);not null" json:"name"`
	ClientID     string                `gorm:"column:client_id;type:varchar(256);not null" json:"client_id"`
	ClientSecret string                `gorm:"column:client_secret;type:varchar(256);not null" json:"-"`
	RedirectURL  string                `gorm:"column:redirect_url;type:varchar(512);not null" json:"redirect_url"`
	AuthURL      string                `gorm:"column:auth_url;type:varchar(512)" json:"auth_url"`
	TokenURL     string                `gorm:"column:token_url;type:varchar(512)" json:"token_url"`
	UserInfoURL  string                `gorm:"column:user_info_url;type:varchar(512)" json:"user_info_url"`
	Scopes       string                `gorm:"column:scopes;type:varchar(512)" json:"scopes"`
	Status       OAuth2ProviderStatus  `gorm:"column:status;type:varchar(32);default:'enabled';index:idx_oauth2_provider_status" json:"status"`
	Sort         int                   `gorm:"column:sort;type:int;default:0" json:"sort"`
	CreatedAt    time.Time             `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_oauth2_provider_deleted_at" json:"-"`
}

// TableName 指定表名
func (*OAuth2Provider) TableName() string {
	return "oauth2_providers"
}

// BeforeCreate 创建前钩子，生成雪花ID
func (p *OAuth2Provider) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		node, err := snowflake.NewNode(1)
		if err != nil {
			return err
		}
		p.ID = node.Generate().Int64()
	}
	return nil
}

// IsAvailable 检查提供商是否可用
func (p *OAuth2Provider) IsAvailable() bool {
	return p.Status == OAuth2ProviderStatusEnabled
}

// UserOAuth2Status 用户绑定状态
type UserOAuth2Status string

const (
	UserOAuth2StatusActive  UserOAuth2Status = "active"  // 活跃
	UserOAuth2StatusRevoked UserOAuth2Status = "revoked" // 解绑
)

// UserOAuth2 用户第三方OAuth2绑定
type UserOAuth2 struct {
	ID             int64                 `gorm:"primaryKey;column:id" json:"id"`
	UserID         int64                 `gorm:"column:user_id;type:bigint;not null;index:idx_user_oauth2_user" json:"user_id"`
	ProviderID     int64                 `gorm:"column:provider_id;type:bigint;not null;index:idx_user_oauth2_provider" json:"provider_id"`
	ProviderType   OAuth2ProviderType    `gorm:"column:provider_type;type:varchar(32);not null" json:"provider_type"`
	OpenID         string                `gorm:"column:open_id;type:varchar(256);not null;index:idx_user_oauth2_openid" json:"open_id"`
	UnionID        string                `gorm:"column:union_id;type:varchar(256);index:idx_user_oauth2_unionid" json:"union_id,omitempty"`
	AccessToken    string                `gorm:"column:access_token;type:varchar(512)" json:"-"`
	RefreshToken   string                `gorm:"column:refresh_token;type:varchar(512)" json:"-"`
	TokenExpiresAt *time.Time            `gorm:"column:token_expires_at;type:timestamp" json:"token_expires_at,omitempty"`
	Nickname       string                `gorm:"column:nickname;type:varchar(128)" json:"nickname"`
	Avatar         string                `gorm:"column:avatar;type:varchar(512)" json:"avatar"`
	Email          string                `gorm:"column:email;type:varchar(128)" json:"email"`
	Status         UserOAuth2Status      `gorm:"column:status;type:varchar(32);default:'active';index:idx_user_oauth2_status" json:"status"`
	LastLoginAt    *time.Time            `gorm:"column:last_login_at;type:timestamp" json:"last_login_at,omitempty"`
	CreatedAt      time.Time             `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_user_oauth2_deleted_at" json:"-"`

	// 关联关系
	User     models.User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Provider OAuth2Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

// TableName 指定表名
func (*UserOAuth2) TableName() string {
	return "user_oauth2"
}

// BeforeCreate 创建前钩子，生成雪花ID
func (u *UserOAuth2) BeforeCreate(tx *gorm.DB) error {
	if u.ID == 0 {
		node, err := snowflake.NewNode(1)
		if err != nil {
			return err
		}
		u.ID = node.Generate().Int64()
	}
	return nil
}

// IsAvailable 检查绑定是否可用
func (u *UserOAuth2) IsAvailable() bool {
	return u.Status == UserOAuth2StatusActive
}

// RecordLogin 记录登录
func (u *UserOAuth2) RecordLogin() {
	now := time.Now()
	u.LastLoginAt = &now
}

// UpdateToken 更新Token
func (u *UserOAuth2) UpdateToken(accessToken, refreshToken string, expiresAt *time.Time) {
	u.AccessToken = accessToken
	u.RefreshToken = refreshToken
	u.TokenExpiresAt = expiresAt
}
