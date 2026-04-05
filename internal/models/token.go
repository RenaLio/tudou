package models

import (
	"database/sql/driver"

	"time"

	"github.com/goccy/go-json"
	"gorm.io/plugin/soft_delete"
)

// TokenStatus 令牌状态
type TokenStatus string

const (
	TokenStatusEnabled  TokenStatus = "enabled"  // 启用
	TokenStatusDisabled TokenStatus = "disabled" // 禁用
	TokenStatusExpired  TokenStatus = "expired"  // 过期
)

// TokenSettings 令牌配置
type TokenSettings struct {
	RateLimit     int      `json:"rateLimit,omitempty"`     // 速率限制（请求/分钟）
	DailyQuota    int64    `json:"dailyQuota,omitempty"`    // 每日配额（token数），未来扩展
	WeeklyQuota   int64    `json:"weeklyQuota,omitempty"`   // 每周配额（token数），未来扩展
	MonthlyQuota  int64    `json:"monthlyQuota,omitempty"`  // 每月配额（token数），未来扩展
	AllowedModels []string `json:"allowedModels,omitempty"` // 允许的模型列表
	IPWhitelist   []string `json:"ipWhitelist,omitempty"`   // IP白名单
}

// Value 实现 driver.Valuer 接口
func (s *TokenSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *TokenSettings) Scan(value interface{}) error {
	if value == nil {
		*s = TokenSettings{}
		return nil
	}
	return unmarshalJSONValue(value, s)
}

// Token 令牌模型
type Token struct {
	ID                  int64                 `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	UserID              int64                 `gorm:"column:user_id;type:bigint;not null;index:idx_token_user" json:"userID,string"`
	GroupID             int64                 `gorm:"column:group_id;type:bigint;not null;index:idx_token_group" json:"groupID,string"`
	Token               string                `gorm:"column:token;type:varchar(512);not null;uniqueIndex:idx_token_value" json:"token"`
	Name                string                `gorm:"column:name;type:varchar(128)" json:"name"`
	Status              TokenStatus           `gorm:"column:status;type:varchar(32);default:'enabled';index:idx_token_status" json:"status"`
	LimitMicros         int64                 `gorm:"column:limit_micros;type:bigint;default:-1;comment:用量限制，单位 micros" json:"limitMicros"` // 用量限制，<0表示不限制，单位 micros
	ExpiresAt           *time.Time            `gorm:"column:expires_at;type:timestamp" json:"expiresAt,omitempty"`
	LoadBalanceStrategy LoadBalanceStrategy   `gorm:"column:load_balance_strategy;type:varchar(32);default:'performance'" json:"loadBalanceStrategy"`
	Settings            TokenSettings         `gorm:"column:settings;type:json" json:"settings"`
	CreatedAt           time.Time             `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt           soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_token_deleted_at" json:"-"`

	// 关联关系
	User  User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Group ChannelGroup `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Stats TokenStats   `gorm:"foreignKey:TokenID;references:ID" json:"stats,omitempty"`
}

// TableName 指定表名
func (*Token) TableName() string {
	return "tokens"
}

// IsAvailable 检查令牌是否可用
func (t *Token) IsAvailable() bool {
	if t.Status != TokenStatusEnabled {
		return false
	}
	if t.IsExpired() {
		return false
	}
	if !t.CheckLimit() {
		return false
	}
	return true
}

// IsExpired 检查令牌是否过期
func (t *Token) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return t.ExpiresAt.Before(time.Now())
}

// CheckLimit 检查用量限制
func (t *Token) CheckLimit() bool {
	if t.LimitMicros < 0 {
		return true
	}
	if t.LimitMicros <= t.Stats.TotalCostMicros {
		return false
	}
	return true
}

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy string

const (
	LoadBalanceStrategyRandom       LoadBalanceStrategy = "random"        // 随机
	LoadBalanceStrategyPerformance  LoadBalanceStrategy = "performance"   // 综合性能优先
	LoadBalanceStrategyTTFTFirst    LoadBalanceStrategy = "ttft_first"    // 响应时间优先优先
	LoadBalanceStrategySuccessFirst LoadBalanceStrategy = "success_first" // 成功率优先
	LoadBalanceStrategyCostFirst    LoadBalanceStrategy = "cost_first"    // 成本优先
	LoadBalanceStrategyTPSFirst     LoadBalanceStrategy = "tps_first"     // TPS优先优先
	LoadBalanceStrategyWeighted     LoadBalanceStrategy = "weighted"      // 加权
	LoadBalanceStrategyLeastConn    LoadBalanceStrategy = "least_conn"    // 最少连接
)

type TokenStats struct {
	TokenID                   int64 `gorm:"column:token_id;primaryKey;uniqueIndex" json:"tokenID"`
	InputToken                int64 `gorm:"column:input_token;type:bigint;default:0;comment:输入token数" json:"inputToken"`
	OutputToken               int64 `gorm:"column:output_token;type:bigint;default:0;comment:输出token数" json:"outputToken"`
	CachedCreationInputTokens int64 `gorm:"column:cached_creation_input_tokens;type:bigint;default:0;comment:缓存创建输入token数" json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64 `gorm:"column:cached_read_input_tokens;type:bigint;default:0;comment:缓存读取输入token数" json:"cachedReadInputTokens"`
	RequestSuccess            int64 `gorm:"column:request_success;type:bigint;default:0;comment:请求成功数" json:"requestSuccess"`
	RequestFailed             int64 `gorm:"column:request_failed;type:bigint;default:0;comment:请求失败数" json:"requestFailed"`
	TotalCostMicros           int64 `gorm:"column:total_cost_micros;type:bigint;default:0;comment:总成本，单位 micros" json:"totalCostMicros"`
}
