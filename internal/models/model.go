package models

import (
	"database/sql/driver"

	"time"

	"github.com/goccy/go-json"
	"gorm.io/plugin/soft_delete"
)

// AIModel AI模型定义
type AIModel struct {
	ID           int64                 `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	Name         string                `gorm:"column:name;type:varchar(128);not null;uniqueIndex:idx_model_name" json:"name"`
	Type         ModelType             `gorm:"column:type;type:varchar(32);default:'chat';index:idx_model_type" json:"type"`
	Description  string                `gorm:"column:description;type:text" json:"description"`
	Pricing      ModelPricing          `gorm:"column:pricing;type:json" json:"pricing"`
	Capabilities ModelCapabilities     `gorm:"column:capabilities;type:json" json:"capabilities"`
	PricingType  ModelPricingType      `gorm:"column:pricing_type;type:varchar(32);default:'tokens';index:idx_model_pricing_type" json:"pricingType"`
	IsEnabled    bool                  `gorm:"column:is_enabled;type:boolean;default:true;index:idx_model_enabled" json:"isEnabled"` // 未来扩展
	Extra        AIModelExtra          `gorm:"column:extra;type:json" json:"extra"`
	CreatedAt    time.Time             `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time             `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt    soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint;index:idx_model_deleted_at;uniqueIndex:idx_model_name" json:"-"`
}

// TableName 指定表名
func (*AIModel) TableName() string {
	return "models"
}

// HasFeature 检查是否支持某特性
func (m *AIModel) HasFeature(feature ModelFeature) bool {
	for _, f := range m.Capabilities.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// CalculateByTokensMicros 按量计费（基于 tokens，返回 micros）
func (m *AIModel) CalculateByTokensMicros(inputTokens, outputTokens int64) int64 {
	return m.CalculateByTokensWithCacheMicros(inputTokens, outputTokens, 0, 0)
}

// CalculateByTokensWithCacheMicros 按量计费（含缓存 tokens，返回 micros）
func (m *AIModel) CalculateByTokensWithCacheMicros(inputTokens, outputTokens, cacheCreateTokens, cacheReadTokens int64) int64 {
	inputCost := (inputTokens - cacheCreateTokens - cacheReadTokens) * pricingPerMillionToMicros(m.Pricing.InputPrice) / pricingTokenUnit
	outputCost := outputTokens * pricingPerMillionToMicros(m.Pricing.OutputPrice) / pricingTokenUnit
	cacheCreateCost := cacheCreateTokens * pricingPerMillionToMicros(m.Pricing.CacheCreatePrice) / pricingTokenUnit
	cacheReadCost := cacheReadTokens * pricingPerMillionToMicros(m.Pricing.CacheReadPrice) / pricingTokenUnit
	return inputCost + outputCost + cacheCreateCost + cacheReadCost
}

// CalculateByTokensDetailedMicros 按量计费（返回详细 breakdown，单位 micros）
func (m *AIModel) CalculateByTokensDetailedMicros(inputTokens, outputTokens, cacheCreateTokens, cacheReadTokens int64) map[string]int64 {
	inputCost := inputTokens * pricingPerMillionToMicros(m.Pricing.InputPrice) / pricingTokenUnit
	outputCost := outputTokens * pricingPerMillionToMicros(m.Pricing.OutputPrice) / pricingTokenUnit
	cacheCreateCost := cacheCreateTokens * pricingPerMillionToMicros(m.Pricing.CacheCreatePrice) / pricingTokenUnit
	cacheReadCost := cacheReadTokens * pricingPerMillionToMicros(m.Pricing.CacheReadPrice) / pricingTokenUnit
	total := inputCost + outputCost + cacheCreateCost + cacheReadCost

	return map[string]int64{
		"input":        inputCost,
		"output":       outputCost,
		"cache_create": cacheCreateCost,
		"cache_read":   cacheReadCost,
		"total":        total,
	}
}

// CalculateByRequestMicros 按次计费（返回 micros）
func (m *AIModel) CalculateByRequestMicros() int64 {
	return moneyFloatToMicros(m.Pricing.PerRequestPrice)
}

// CalculateByRequestDetailedMicros 按次计费（返回详细 breakdown，单位 micros）
func (m *AIModel) CalculateByRequestDetailedMicros() map[string]int64 {
	requestCost := m.CalculateByRequestMicros()
	return map[string]int64{
		"request": requestCost,
		"total":   requestCost,
	}
}

type ModelPricingType string

const (
	ModelPricingTypeTokens  ModelPricingType = "tokens"  // 按 token 计费
	ModelPricingTypeRequest ModelPricingType = "request" // 按次计费
)

// ModelType 模型类型
type ModelType string

const (
	ModelTypeChat      ModelType = "chat"      // 聊天模型
	ModelTypeEmbedding ModelType = "embedding" // 嵌入模型
	ModelTypeImage     ModelType = "image"     // 图像模型
	ModelTypeAudio     ModelType = "audio"     // 音频模型
	ModelTypeMulti     ModelType = "multi"     // 多模态模型
)

// ModelFeature 模型特性
type ModelFeature string

const (
	ModelFeatureStream    ModelFeature = "stream"    // 支持流式
	ModelFeatureVision    ModelFeature = "vision"    // 支持视觉
	ModelFeatureFunction  ModelFeature = "function"  // 支持函数调用
	ModelFeatureJSON      ModelFeature = "json"      // 支持JSON模式
	ModelFeatureReasoning ModelFeature = "reasoning" // 支持推理
)

// ModelPricing 模型定价信息
type ModelPricing struct {
	InputPrice       float64 `json:"inputPrice"`       // 输入价格 (per 1M tokens)
	OutputPrice      float64 `json:"outputPrice"`      // 输出价格 (per 1M tokens)
	CacheCreatePrice float64 `json:"cacheCreatePrice"` // 缓存创建价格 (per 1M tokens)
	CacheReadPrice   float64 `json:"cacheReadPrice"`   // 缓存读取价格 (per 1M tokens)
	PerRequestPrice  float64 `json:"perRequestPrice"`  // 按次计费价格 (per request)
}

// Value 实现 driver.Valuer 接口
func (p ModelPricing) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan 实现 sql.Scanner 接口
func (p *ModelPricing) Scan(value interface{}) error {
	if value == nil {
		*p = ModelPricing{}
		return nil
	}
	return unmarshalJSONValue(value, p)
}

// ModelCapabilities 模型能力配置
type ModelCapabilities struct {
	InputLimit       int            `json:"inputLimit,omitempty"`       // 输入上下文长度限制
	MaxTokens        int            `json:"maxTokens,omitempty"`        // 最大上下文长度
	MaxOutputTokens  int            `json:"maxOutputTokens,omitempty"`  // 最大输出长度
	Features         []ModelFeature `json:"features,omitempty"`         // 支持的特性
	SupportedFormats []string       `json:"supportedFormats,omitempty"` // 支持的格式
}

// Value 实现 driver.Valuer 接口
func (c ModelCapabilities) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner 接口
func (c *ModelCapabilities) Scan(value interface{}) error {
	if value == nil {
		*c = ModelCapabilities{}
		return nil
	}
	return unmarshalJSONValue(value, c)
}

type AIModelExtra struct {
	SyncModelInfoPath string `json:"syncModelInfoPath,omitempty"` // 格式：@提供商/模型id，用于向类似models.dev Web API同步模型信息
}

func (e AIModelExtra) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func (e *AIModelExtra) Scan(value interface{}) error {
	if value == nil {
		*e = AIModelExtra{}
		return nil
	}
	return unmarshalJSONValue(value, e)
}
