package types

import "time"

type ResponseMetrics struct {
	Provider string
	Model    string
	Format   Format
	IsStream bool
	Status   int // 状态码 0-未初始化 1-成功 2-失败

	DNSTime      time.Duration // DNS 解析耗时
	TCPTime      time.Duration // TCP 连接耗时
	TLSTime      time.Duration // TLS 握手耗时
	TTFB         time.Duration // 首字时间
	TransferTime time.Duration // 数据传输耗时
	TotalTime    time.Duration // 总耗时

	TTFT time.Duration // 首token时间

	Usage Usage
	Extra map[string]any
}

type MetricsCallback func(metrics *ResponseMetrics)

type (
	Usage struct {
		InputTokens               int64
		OutputTokens              int64
		CachedTokens              int64
		CachedCreationInputTokens int64
		CachedReadInputTokens     int64
		ReasoningTokens           int64
		TotalTokens               int64
	}
)
