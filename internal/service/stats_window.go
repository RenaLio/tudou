package service

import (
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
)

const (
	window3HDuration = 3 * time.Hour                          // 观测窗口总时长为3小时
	bucketDuration   = 15 * time.Minute                       // 每个桶的粒度为15分钟
	bucketCount      = int(window3HDuration / bucketDuration) // 窗口内桶的数量（3小时 / 15分钟 = 12）
)

// bucketAccumulator 临时累加器，用于将一条条请求日志聚合到对应时间桶的统计数据
type bucketAccumulator struct {
	inputToken                int64 // 输入token总量
	outputToken               int64 // 输出token总量
	cachedCreationInputTokens int64 // 缓存创建时涉及的输入token量
	cachedReadInputTokens     int64 // 缓存读取时涉及的输入token量
	requestSuccess            int64 // 成功请求数
	requestFailed             int64 // 失败请求数
	totalCostMicros           int64 // 总消耗（微单位，此处为微元或微美元等）
	ttftSum                   int64 // TTFT（首token时间）求和
	ttftCount                 int64 // 有效TTFT的计数（用于计算平均值）
	transferTimeSum           int64 // 传输耗时求和（用于计算平均TPS）
}

// buildObservationWindow3H 根据当前时间和请求日志，构建一个3小时观测窗口（内部分为15分钟桶）
func buildObservationWindow3H(now time.Time, logs []*models.RequestLog) models.ObservationWindow3H {
	// 计算窗口的起止时间（基于传入的 now 对齐桶边界后得出）
	windowStart, windowEnd := observationWindowRange(now)

	// 初始化一个空的观测窗口对象，并预先创建12个桶（每个15分钟）
	window := models.NewObservationWindow3H()
	window.Buckets = make([]models.ObservationBucket15M, bucketCount) // 分配桶切片
	// 初始化每个桶的 StartAt 和 EndAt 时间
	for i := 0; i < bucketCount; i++ {
		startAt := windowStart.Add(time.Duration(i) * bucketDuration) // 第i个桶的开始时间
		window.Buckets[i] = models.ObservationBucket15M{
			StartAt: startAt,
			EndAt:   startAt.Add(bucketDuration), // 桶结束时间 = 开始时间 + 15分钟
		}
	}

	// 创建与桶数量相同的累加器切片，用于聚合统计数据
	acc := make([]bucketAccumulator, bucketCount)

	// 遍历所有请求日志，将每条日志归入对应的时间桶
	for _, log := range logs {
		if log == nil {
			continue // 跳过空日志
		}
		createdAt := log.CreatedAt.UTC() // 统一使用UTC时间比较
		// 如果日志时间不在 [windowStart, windowEnd) 范围内，则忽略这条日志
		if createdAt.Before(windowStart) || !createdAt.Before(windowEnd) {
			continue
		}
		// 根据日志时间计算它应该落入的桶索引
		idx := int(createdAt.Sub(windowStart) / bucketDuration)
		// 防止计算出的索引越界（理论上不会发生，但做安全保护）
		if idx < 0 || idx >= bucketCount {
			continue
		}

		// 获取对应桶的累加器指针，方便直接累加
		item := &acc[idx]
		// 累加各类指标
		item.inputToken += log.InputToken
		item.outputToken += log.OutputToken
		item.cachedCreationInputTokens += log.CachedCreationInputTokens
		item.cachedReadInputTokens += log.CachedReadInputTokens
		item.totalCostMicros += log.CostMicros
		item.transferTimeSum += log.TransferTime
		// TTFT 大于 0 才算有效值，参与平均值计算
		if log.TTFT > 0 {
			item.ttftSum += log.TTFT
			item.ttftCount++
		}
		// 按请求状态统计成功/失败次数
		switch log.Status {
		case models.RequestStatusSuccess:
			item.requestSuccess++
		case models.RequestStatusFail:
			item.requestFailed++
		default:
			item.requestFailed++
		}
	}

	// 将累加器中的聚合结果写入对应桶的最终输出结构
	for i := 0; i < bucketCount; i++ {
		b := acc[i]
		bucket := &window.Buckets[i]
		// 直接赋值基础累加指标
		bucket.InputToken = b.inputToken
		bucket.OutputToken = b.outputToken
		bucket.CachedCreationInputTokens = b.cachedCreationInputTokens
		bucket.CachedReadInputTokens = b.cachedReadInputTokens
		bucket.RequestSuccess = b.requestSuccess
		bucket.RequestFailed = b.requestFailed
		bucket.TotalCostMicros = b.totalCostMicros
		// 如果存在有效TTFT记录，计算平均TTFT（整数结果）
		if b.ttftCount > 0 {
			bucket.AvgTTFT = int(b.ttftSum / b.ttftCount)
		}
		// 如果有传输耗时且输出token不为0，计算平均TPS（tokens per second）
		// 公式：输出token * 1000 / 传输耗时（毫秒） -> 得到每秒token数
		if b.transferTimeSum > 0 {
			bucket.AvgTPS = float64(b.outputToken) * 1000 / float64(b.transferTimeSum)
		}
	}

	return window // 返回构建好的3小时观测窗口
}

// observationWindowRange 根据当前时间计算3小时观测窗口的 [开始时间, 结束时间)
// 窗口结束时间为当前时间所在桶的结束时间，开始时间为结束时间向前推3小时
func observationWindowRange(now time.Time) (time.Time, time.Time) {
	// 将当前时间向下对齐到15分钟桶的起点（如 10:07 对齐到 10:00）
	alignedBucketStart := now.UTC().Truncate(bucketDuration)
	// 窗口开始时间 = 对齐后的桶起点 - (12-1) * 15分钟 = 对齐桶起点 - 2小时45分钟
	// 这样得到的窗口正好包含12个完整桶（如 [07:00, 10:00) 共3小时）
	windowStart := alignedBucketStart.Add(-time.Duration(bucketCount-1) * bucketDuration)
	// 窗口结束时间 = 对齐后的桶起点 + 15分钟，即当前桶的结束时间（左闭右开）
	windowEnd := alignedBucketStart.Add(bucketDuration)
	return windowStart, windowEnd
}

// toObservationWindow3HResponse 将领域模型 ObservationWindow3H 转换为 API 响应模型
func toObservationWindow3HResponse(window models.ObservationWindow3H) v1.ObservationWindow3HResponse {
	// 初始化响应对象，并复制窗口级别基础字段
	resp := v1.ObservationWindow3HResponse{
		WindowMinutes: window.WindowMinutes,                                            // 窗口总分钟数（通常为180）
		BucketMinutes: window.BucketMinutes,                                            // 单个桶分钟数（通常为15）
		Buckets:       make([]v1.ObservationBucket15MResponse, 0, len(window.Buckets)), // 预分配桶切片
	}
	// 遍历源窗口的每个桶，逐个转换并添加到响应中
	for _, b := range window.Buckets {
		resp.Buckets = append(resp.Buckets, v1.ObservationBucket15MResponse{
			StartAt:                   b.StartAt,                   // 桶开始时间
			EndAt:                     b.EndAt,                     // 桶结束时间
			InputToken:                b.InputToken,                // 输入token
			OutputToken:               b.OutputToken,               // 输出token
			CachedCreationInputTokens: b.CachedCreationInputTokens, // 缓存创建输入token
			CachedReadInputTokens:     b.CachedReadInputTokens,     // 缓存读取输入token
			RequestSuccess:            b.RequestSuccess,            // 成功请求数
			RequestFailed:             b.RequestFailed,             // 失败请求数
			TotalCostMicros:           b.TotalCostMicros,           // 总消耗 (微单位)
			AvgTTFT:                   b.AvgTTFT,                   // 平均首token时间
			AvgTPS:                    b.AvgTPS,                    // 平均输出tokens/秒
		})
	}
	return resp // 返回转换后的响应
}
