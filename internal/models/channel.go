package models

import (
	"database/sql/driver"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"gorm.io/plugin/soft_delete"
)

// Channel 渠道模型
type Channel struct {
	ID          int64                 `gorm:"primaryKey;column:id;type:bigint;not null;autoIncrement:false" json:"id,string"`
	Type        ChannelType           `gorm:"column:type;type:varchar(32);not null;index:idx_channel_type" json:"type"`
	Name        string                `gorm:"column:name;type:varchar(128);not null;index:idx_channel_name" json:"name"`
	BaseURL     string                `gorm:"column:base_url;type:varchar(512);not null" json:"baseURL"`
	APIKey      string                `gorm:"column:api_key;type:varchar(512);not null" json:"apiKey"`
	Weight      int                   `gorm:"column:weight;type:int;default:100" json:"weight"`
	Status      ChannelStatus         `gorm:"column:status;type:varchar(16);default:'enabled';comment:渠道状态" json:"status"`
	Remark      string                `gorm:"column:remark;type:text;comment:渠道备注" json:"remark"`
	Tag         string                `gorm:"column:tag;type:varchar(128);comment:渠道标签" json:"tag"`
	Model       string                `gorm:"column:model;type:text" json:"model"`              // 支持的模型列表，逗号分隔，用于同步
	CustomModel string                `gorm:"column:custom_model;type:text" json:"customModel"` // 自定义模型列表，逗号分隔，不同步
	Settings    ChannelSettings       `gorm:"column:settings;type:json;comment:渠道配置" json:"settings"`
	Extra       ChannelExtra          `gorm:"column:extra;type:json;comment:渠道扩展信息" json:"extra"`
	PriceRate   float64               `gorm:"column:price_rate;type:decimal(20,6);default:1;comment:渠道价格比例" json:"priceRate"`
	ExpiredAt   *time.Time            `gorm:"column:expired_at;comment:渠道过期时间" json:"expiredAt"`
	CreatedAt   time.Time             `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt   soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_channel_deleted_at" json:"-"`

	//MaxConnections int                   `gorm:"column:max_connections;type:int;default:0;comment:最大连接数" json:"maxConnections"`
	// 关联关系
	Groups []ChannelGroup `gorm:"many2many:group_channels;foreignKey:ID;joinForeignKey:ChannelID;References:ID;joinReferences:GroupID" json:"groups,omitempty"`
	Stats  *ChannelStats  `gorm:"foreignKey:ChannelID;references:ID" json:"stats,omitempty"`
}

// TableName 指定表名
func (*Channel) TableName() string {
	return "channels"
}

// IsAvailable 检查渠道是否可用
func (c *Channel) IsAvailable() bool {
	if c.Status != ChannelStatusEnabled {
		return false
	}
	// 如果没有设置过期时间，或者过期时间在当前时间之后，则可用
	return c.ExpiredAt == nil || c.ExpiredAt.After(time.Now())
}

func (c *Channel) SupportModel(model string) bool {
	model = strings.TrimSpace(model)

	modelsMap := c.Models()
	_, exists := modelsMap[model]

	return exists
}

// Models 获取渠道支持的模型列表 ，key为CallModel，value为upstreamModel
func (c *Channel) Models() map[string]string {
	modelMap := make(map[string]string)
	models := strings.Split(c.Model, ",")
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		modelMap[m] = m
	}
	// 自定义模型
	customModels := strings.Split(c.CustomModel, ",")
	for _, m := range customModels {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		modelMap[m] = m
	}
	for k, v := range c.Extra.ModelMappings {
		modelMap[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return modelMap
}

// ChannelType 渠道类型
type ChannelType string

const (
	ChannelTypeOpenAI ChannelType = "openai" // OpenAI
	ChannelTypeClaude ChannelType = "claude" // Anthropic Claude
	ChannelTypeAzure  ChannelType = "azure"  // Azure OpenAI
	ChannelTypeCustom ChannelType = "custom" // 自定义
)

// ChannelStatus 渠道状态
type ChannelStatus string

const (
	ChannelStatusEnabled  ChannelStatus = "enabled"  // 启用
	ChannelStatusDisabled ChannelStatus = "disabled" // 禁用
	ChannelStatusExpired  ChannelStatus = "expired"  // 过期
)

// ChannelSettings 渠道配置JSON字段
type ChannelSettings struct {
	Timeout            int     `json:"timeout,omitempty"`            // 请求超时(秒)
	MaxRetries         int     `json:"maxRetries,omitempty"`         // 最大重试次数
	RetryInterval      int     `json:"retryInterval,omitempty"`      // 重试间隔(毫秒)
	EnableStream       bool    `json:"enableStream,omitempty"`       // 是否支持流式
	StreamTimeout      int     `json:"streamTimeout,omitempty"`      // 流式超时(秒)
	MaxTokens          int     `json:"maxTokens,omitempty"`          // 最大token限制
	DefaultTemperature float32 `json:"defaultTemperature,omitempty"` // 默认温度
	CircuitThreshold   int     `json:"circuitThreshold,omitempty"`   // 熔断阈值(连续失败次数)
	CircuitTimeout     int     `json:"circuitTimeout,omitempty"`     // 熔断恢复时间(秒)
	MaxConcurrent      int     `json:"maxConcurrent,omitempty"`      // 最大并发数
}

// Value 实现 driver.Valuer 接口
func (s ChannelSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *ChannelSettings) Scan(value interface{}) error {
	if value == nil {
		*s = ChannelSettings{}
		return nil
	}
	return unmarshalJSONValue(value, s)
}

// ChannelExtra 渠道扩展信息JSON字段
type ChannelExtra struct {
	Headers       map[string]string `json:"headers,omitempty"`       // 自定义请求头
	Description   string            `json:"description,omitempty"`   // 详细描述
	DocsURL       string            `json:"docsURL,omitempty"`       // 文档链接
	Region        string            `json:"region,omitempty"`        // 区域
	Tier          string            `json:"tier,omitempty"`          // 服务等级
	ModelMappings map[string]string `json:"modelMappings,omitempty"` // 模型名称映射（调用名(call_model) -> 计费名(upstream_model)）
}

// Value 实现 driver.Valuer 接口
func (e ChannelExtra) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Scan 实现 sql.Scanner 接口
func (e *ChannelExtra) Scan(value interface{}) error {
	if value == nil {
		*e = ChannelExtra{}
		return nil
	}
	return unmarshalJSONValue(value, e)
}

// ChannelStats 渠道统计信息 仅用于看板统计显示
type ChannelStats struct {
	ChannelID                 int64               `gorm:"column:channel_id;primaryKey;index:idx_channel_stats_channel" json:"channelID,string"`
	InputToken                int64               `gorm:"column:input_token;type:bigint;default:0;comment:输入token数" json:"inputToken"`
	OutputToken               int64               `gorm:"column:output_token;type:bigint;default:0;comment:输出token数" json:"outputToken"`
	CachedCreationInputTokens int64               `gorm:"column:cached_creation_input_tokens;type:bigint;default:0;comment:缓存创建输入token数" json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64               `gorm:"column:cached_read_input_tokens;type:bigint;default:0;comment:缓存读取输入token数" json:"cachedReadInputTokens"`
	RequestSuccess            int64               `gorm:"column:request_success;type:bigint;default:0;comment:请求成功数" json:"requestSuccess"`
	RequestFailed             int64               `gorm:"column:request_failed;type:bigint;default:0;comment:请求失败数" json:"requestFailed"`
	TotalCostMicros           int64               `gorm:"column:total_cost_micros;type:bigint;default:0;comment:总成本，单位 micros" json:"totalCostMicros"`
	AvgTTFT                   int                 `gorm:"column:avg_ttft;type:int;default:0;comment:平均首字延迟(ms)" json:"avgTTFT"`
	AvgTPS                    float64             `gorm:"column:avg_tps;type:decimal(8,2);default:0;comment:平均每秒吐字" json:"avgTPS"`
	Window3H                  ObservationWindow3H `gorm:"column:window_3h;type:json" json:"window3h"`
}

// ChannelModelStats 渠道模型统计信息 仅用于看板统计显示
type ChannelModelStats struct {
	ChannelID                 int64               `gorm:"column:channel_id;primaryKey;index:idx_channel_model_stats_channel_model" json:"channelID,string"`
	Model                     string              `gorm:"column:model;type:varchar(128);primaryKey;not null;index:idx_channel_model_stats_channel_model" json:"model"`
	InputToken                int64               `gorm:"column:input_token;type:bigint;default:0;comment:输入token数" json:"inputToken"`
	OutputToken               int64               `gorm:"column:output_token;type:bigint;default:0;comment:输出token数" json:"outputToken"`
	CachedCreationInputTokens int64               `gorm:"column:cached_creation_input_tokens;type:bigint;default:0;comment:缓存创建输入token数" json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64               `gorm:"column:cached_read_input_tokens;type:bigint;default:0;comment:缓存读取输入token数" json:"cachedReadInputTokens"`
	RequestSuccess            int64               `gorm:"column:request_success;type:bigint;default:0;comment:请求成功数" json:"requestSuccess"`
	RequestFailed             int64               `gorm:"column:request_failed;type:bigint;default:0;comment:请求失败数" json:"requestFailed"`
	TotalCostMicros           int64               `gorm:"column:total_cost_micros;type:bigint;default:0;comment:总成本，单位 micros" json:"totalCostMicros"`
	AvgTTFT                   int                 `gorm:"column:avg_ttft;type:int;default:0;comment:平均首字延迟(ms)" json:"avgTTFT"`
	AvgTPS                    float64             `gorm:"column:avg_tps;type:decimal(8,2);default:0;comment:平均每秒吐字" json:"avgTPS"`
	Window3H                  ObservationWindow3H `gorm:"column:window_3h;type:json" json:"window3h"`
}
