package types

import (
	"time"

	"github.com/goccy/go-json"
)

type ResponseMetrics struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Format     Format `json:"format"`
	IsStream   bool   `json:"isStream"`
	Status     int    `json:"status"` // 状态码 0-未初始化 1-成功 2-失败
	StatusCode int    `json:"statusCode"`

	DNSTime      time.Duration `json:"DNSTime"`      // DNS 解析耗时
	TCPTime      time.Duration `json:""`             // TCP 连接耗时
	TLSTime      time.Duration `json:"TLSTime"`      // TLS 握手耗时
	TTFB         time.Duration `json:"TTFB"`         // 首字时间
	TransferTime time.Duration `json:"TransferTime"` // 数据传输耗时
	TotalTime    time.Duration `json:"totalTime"`    // 总耗时

	TTFT time.Duration `json:"TTFT"` // 首token时间

	Usage Usage          `json:"usage"`
	Extra map[string]any `json:"extra"`
}

func (m *ResponseMetrics) String() string {
	json, _ := json.Marshal(m)
	return string(json)
}

type MetricsCallback func(metrics *ResponseMetrics)

type (
	Usage struct {
		InputTokens               int64 // total input (cacheXX is in input)
		OutputTokens              int64 // total output
		CachedTokens              int64
		CachedCreationInputTokens int64
		CachedReadInputTokens     int64
		ReasoningTokens           int64
		TotalTokens               int64
	}
)
