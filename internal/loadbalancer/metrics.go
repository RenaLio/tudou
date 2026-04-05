package loadbalancer

import (
	"sync"
	"time"
)

type Record struct {
	ModelID int64 `json:"modelId"`
	//GroupID          int64   `json:"groupId"`
	//GroupName        string  `json:"groupName"`
	ChannelID   int64  `json:"channelId"`
	ChannelName string `json:"channelName"`
	//ChannelPriceRate float64 `json:"channelPriceRate"`

	//InputTokens               int64   `json:"inputTokens"`
	//InputPrice                float64 `json:"inputPrice"`
	OutputTokens int64 `json:"outputTokens"`
	//OutputPrice               float64 `json:"outputPrice"`
	//CachedCreationInputTokens int64   `json:"cachedCreationInputTokens"`
	//CachedCreationInputPrice  float64 `json:"cachedCreationInputPrice"`
	//CachedReadInputTokens     int64   `json:"cachedReadInputTokens"`
	//CachedReadInputPrice      float64 `json:"cachedReadInputPrice"`
	//Price                     float64 `json:"price"`
	TTFT         int64 `json:"ttft"`         //ms
	TTFB         int64 `json:"ttfb"`         //ms
	TransferTime int64 `json:"transferTime"` //ms
	//RequestId     string `json:"requestId"`
	//RequestFormat string `json:"requestFormat"`
	//TransFormat   string `json:"transFormat"`
	//CreateTime    int64  `json:"createTime"`
	Status int `json:"status"`
	//Error  string `json:"error"`
}

type ChannelModelMetrics struct {
	ChannelID     string `json:"channelId"`
	ModelID       string `json:"modelId"`
	TargetModelID string `json:"targetModelId"`
	Status        int    `json:"status"` // 0: ok, 1: unhealthy 2: circuit

	BaseWeight int64 `json:"baseWeight"`
	AvgTTFT    int64 `json:"avgTTFT"`
	AvgTPS     int64 `json:"avgTPS"`

	RequestCount int64 `json:"requestCount"`
	SuccessCount int64 `json:"successCount"`

	CurrentRetryIndex int `json:"currentRetryIndex"`

	NextRetryTime int64 `json:"nextRetryTime"`

	LastUsedAt int64 `json:"lastUsedAt"`
	mu         sync.RWMutex
}

// Normalize TTFT score to a value between 0 and 1000
func normalizeTTFTScore(num int64) int {
	if num <= 0 {
		return 1000
	}
	if num >= 10000 {
		return 0
	}
	return int((10000 - num) * 1000 / 10000)
}

// Normalize TPS score to a value between 0 and 1000
func normalizeTPSScore(num int64) int {
	if num <= 0 {
		return 0
	}
	if num >= 1000 {
		return 1000
	}
	return int(num * 1000 / 1000)
}

// Endpoint 定义 (对应你的 ChannelModel)
// 这是最小路由单元，包含了 LB 决策和 Proxy 请求所需的一切信息
type Endpoint struct {
	// =====================================
	// 1. 静态配置区 (只读，初始化后不修改)
	// =====================================
	ChannelID     string `json:"channelId"`     // 渠道 ID
	ChannelType   string `json:"channelType"`   // [补充] 极其重要：用于决定协议转换器 (如 openai, bedrock, azure)
	ModelID       string `json:"modelId"`       // 标准模型名 (如 "claude-3-5")
	TargetModelID string `json:"targetModelId"` // 真实请求上游的模型名
	BaseWeight    int64  `json:"baseWeight"`    // 基础静态权重

	// =====================================
	// 2. 动态指标区 (高频并发读写，需加锁)
	// =====================================
	mu sync.RWMutex

	Status int `json:"status"` // 0: 健康, 1: 亚健康(降权), 2: 熔断(Circuit)

	// 性能平滑指标 (必须使用 float64 保证 EMA 精度)
	EmaTTFT        float64 `json:"emaTTFT"`        // 平均首字延迟 (ms)
	EmaTPS         float64 `json:"emaTPS"`         // 平均每秒生成 Token 数
	EmaSuccessRate float64 `json:"emaSuccessRate"` // 近期成功率 (0.0 ~ 100.0)

	// 熔断与恢复相关
	ConsecutiveFails int   `json:"consecutiveFails"` // 连续失败次数，用于快速触发熔断
	NextRetryTime    int64 `json:"nextRetryTime"`    // Unix时间戳：什么时候可以尝试半开恢复(Half-Open)
	LastUsedAt       int64 `json:"lastUsedAt"`       // Unix时间戳：用于探索/利用策略
}

// -------------------------------------------------------------------
// 提供几个安全的并发操作方法 (封装好，不要让外部直接动 mu)
// -------------------------------------------------------------------

// IsAvailable 给 LB 调用，判断该节点当前是否可以被选中
func (e *Endpoint) IsAvailable() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.Status == 0 || e.Status == 1 {
		return true
	}

	// 如果处于熔断状态，检查是否到了下一次探测的时间
	if e.Status == 2 && time.Now().Unix() >= e.NextRetryTime {
		return true // 半开状态，允许放行一个请求去试试
	}

	return false
}

// UpdateMetrics 给 MetricsCollector 调用，异步更新指标
func (e *Endpoint) UpdateMetrics(isSuccess bool, ttft float64, tps float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.LastUsedAt = time.Now().Unix()

	if isSuccess {
		// --- 成功处理逻辑 ---
		// EMA 平滑公式 (例如 0.8 旧 + 0.2 新)
		e.EmaTTFT = (0.8 * e.EmaTTFT) + (0.2 * ttft)
		e.EmaTPS = (0.8 * e.EmaTPS) + (0.2 * tps)
		e.EmaSuccessRate = (0.95 * e.EmaSuccessRate) + (0.05 * 1.0)

		e.ConsecutiveFails = 0
		e.Status = 0 // 恢复健康

	} else {
		// --- 失败处理逻辑 ---
		e.EmaSuccessRate = (0.95 * e.EmaSuccessRate) + (0.05 * 0.0)
		e.ConsecutiveFails++

		// 触发熔断策略 (例如连续失败 3 次)
		if e.ConsecutiveFails >= 3 && e.Status != 2 {
			e.Status = 2
			// 熔断惩罚时间：60秒后才能重试
			e.NextRetryTime = time.Now().Add(60 * time.Second).Unix()
		}
	}
}
