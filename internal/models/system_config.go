package models

import (
	"database/sql/driver"

	"time"

	"github.com/goccy/go-json"
)

// ConfigType 配置类型
type ConfigType string

const (
	ConfigTypeString ConfigType = "string" // 字符串
	ConfigTypeInt    ConfigType = "int"    // 整数
	ConfigTypeFloat  ConfigType = "float"  // 浮点数
	ConfigTypeBool   ConfigType = "bool"   // 布尔值
	ConfigTypeJSON   ConfigType = "json"   // JSON对象
	ConfigTypeArray  ConfigType = "array"  // 数组
)

// ConfigScope 配置作用域
type ConfigScope string

const (
	ConfigScopeSystem ConfigScope = "system" // 系统级
	ConfigScopeUser   ConfigScope = "user"   // 用户级
)

// ConfigValue 配置值封装
type ConfigValue struct {
	Raw       string `json:"-"`
	ValueData any    `json:"value"`
}

// Value 实现 driver.Valuer 接口
func (v *ConfigValue) Value() (driver.Value, error) {
	return json.Marshal(v.ValueData)
}

// Scan 实现 sql.Scanner 接口
func (v *ConfigValue) Scan(value interface{}) error {
	if value == nil {
		v.ValueData = nil
		return nil
	}
	return unmarshalJSONValue(value, &v.ValueData)
}

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          int64       `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	Key         string      `gorm:"column:key;type:varchar(128);not null;uniqueIndex:idx_config_key" json:"key"`
	Value       ConfigValue `gorm:"column:value;type:json" json:"value"`
	Type        ConfigType  `gorm:"column:type;type:varchar(32);default:'string'" json:"type"`
	Scope       ConfigScope `gorm:"column:scope;type:varchar(32);default:'system'" json:"scope"`
	Description string      `gorm:"column:description;type:text" json:"description"`
	IsEditable  bool        `gorm:"column:is_editable;type:boolean;default:true" json:"isEditable"`
	IsVisible   bool        `gorm:"column:is_visible;type:boolean;default:true" json:"isVisible"`
	Sort        int         `gorm:"column:sort;type:int;default:0" json:"sort"`
	CreatedAt   time.Time   `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time   `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName 指定表名
func (*SystemConfig) TableName() string {
	return "system_configs"
}

// GetString 获取字符串值
func (c *SystemConfig) GetString() string {
	if v, ok := c.Value.ValueData.(string); ok {
		return v
	}
	return ""
}

// GetInt 获取整数值
func (c *SystemConfig) GetInt() int64 {
	if v, ok := c.Value.ValueData.(float64); ok {
		return int64(v)
	}
	return 0
}

// GetBool 获取布尔值
func (c *SystemConfig) GetBool() bool {
	if v, ok := c.Value.ValueData.(bool); ok {
		return v
	}
	return false
}

// GetFloat 获取浮点值
func (c *SystemConfig) GetFloat() float64 {
	if v, ok := c.Value.ValueData.(float64); ok {
		return v
	}
	return 0
}

// SetValue 设置值
func (c *SystemConfig) SetValue(val interface{}) {
	c.Value.ValueData = val
}

// ConfigItem 配置项（用于返回）
type ConfigItem struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Type        ConfigType  `json:"type"`
	Description string      `json:"description"`
}

// DefaultConfigs 默认配置
var DefaultConfigs = []SystemConfig{
	{Key: "site.name", Value: ConfigValue{ValueData: "Tudou LLM Gateway"}, Type: ConfigTypeString, Description: "站点名称"},
	{Key: "site.logo", Value: ConfigValue{ValueData: "/logo.png"}, Type: ConfigTypeString, Description: "站点Logo"},
	{Key: "user.register_enabled", Value: ConfigValue{ValueData: true}, Type: ConfigTypeBool, Description: "是否允许注册"},
	{Key: "user.default_quota", Value: ConfigValue{ValueData: 1000000}, Type: ConfigTypeInt, Description: "新用户默认配额"},
	{Key: "request.timeout", Value: ConfigValue{ValueData: 60}, Type: ConfigTypeInt, Description: "请求超时时间(秒)"},
	{Key: "request.max_retries", Value: ConfigValue{ValueData: 3}, Type: ConfigTypeInt, Description: "最大重试次数"},
	{Key: "log.retention_days", Value: ConfigValue{ValueData: 30}, Type: ConfigTypeInt, Description: "日志保留天数"},
}
