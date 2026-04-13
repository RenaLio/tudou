package loadbalancer

import (
	"sync"
	"time"
)

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

func normalizeWeight(weight int64) int {
	if weight <= 0 {
		return 0
	}
	weight = weight * 10
	if weight >= 1000 {
		return 1000
	}
	return int(weight)
}

type Endpoint struct {
	// =====================================
	// 1. 静态配置区 (只读，初始化后不修改)
	// =====================================
	ChannelID     int64   `json:"channelId"`     // 渠道 ID
	ChannelType   string  `json:"channelType"`   // [补充] 极其重要：用于决定协议转换器 (如 openai, bedrock, azure)
	Model         string  `json:"model"`         // 标准模型名 (如 "claude-3-5")
	UpstreamModel string  `json:"upstreamModel"` // 真实请求上游的模型名
	BaseWeight    int64   `json:"baseWeight"`    // 基础静态权重
	CostRate      float64 `json:"costRate"`      // 成本倍率

	// =====================================
	// 2. 动态指标区 (高频并发读写，需加锁)
	// =====================================
	mu sync.RWMutex

	Status int `json:"status"` // 0: 健康, 1: 亚健康(降权), 2: 熔断(Circuit)

	// 性能平滑指标 (必须使用 float64 保证 EMA 精度)
	EmaTTFT        float64 `json:"emaTTFT"`        // 平均首字延迟 (ms)
	EmaTPS         float64 `json:"emaTPS"`         // 平均每秒生成 Token 数
	EmaSuccessRate float64 `json:"emaSuccessRate"` // 近期成功率 (0.0 ~ 1.0)

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
		return true // 半开状态，允许放行一些请求去试试
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
		interval := getMinBackoffInterval()
		// 触发熔断策略
		if e.ConsecutiveFails >= getCircuitBreakThreshold() && e.Status != 2 {
			e.Status = 2
		} else {
			e.Status = 1
		}
		num := min(e.ConsecutiveFails, getCircuitBreakThreshold())
		interval = interval << num
		if interval > getMaxBackoffInterval() {
			interval = getMaxBackoffInterval()
		}
		e.NextRetryTime = time.Now().Add(time.Duration(interval) * time.Millisecond).Unix()
	}
}

func (e *Endpoint) ScoreWithWeights(w PerformanceWeights) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	score := 0.0

	// 成功率得分
	if w.SuccessRate > 0 {
		score += e.EmaSuccessRate * 1000 * w.SuccessRate
	}

	// TTFT得分
	if w.TTFT > 0 {
		score += float64(normalizeTTFTScore(int64(e.EmaTTFT))) * w.TTFT
	}

	// TPS得分
	if w.TPS > 0 {
		score += float64(normalizeTPSScore(int64(e.EmaTPS))) * w.TPS
	}

	if w.Cost > 0 {
		// 假设 e.CostRate 正常是 1.0, 便宜的是 0.2, 贵的是 2.0
		// 我们用 1000 / CostRate 来算分 (限制一下下限防止除零)
		safeCostRate := e.CostRate
		if safeCostRate <= 0.01 {
			safeCostRate = 0.01 // 防止倍率填错导致无限大
		}
		// 以倍率 1.0 为基准得 1000 分，倍率 0.2 得 5000 分
		costScore := 1000.0 / safeCostRate
		if costScore > 1000 {
			costScore = 1000
		}
		score += costScore * w.Cost
	}

	// 基础权重
	score += float64(normalizeWeight(e.BaseWeight)) * w.weight

	if e.Status == 1 {
		// 降权
		for _ = range e.ConsecutiveFails {
			score *= 0.96
		}
	}

	return score
}
