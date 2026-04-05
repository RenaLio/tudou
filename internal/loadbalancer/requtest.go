package loadbalancer

import "github.com/RenaLio/tudou/internal/models"

type Request struct {
	GroupID int64  `json:"groupId"`
	UserID  int64  `json:"userId"`
	Model   string `json:"model"`
	// 一致性哈希路由，增加缓存命中概率
	CacheKey string `json:"cacheKey"`
	// 当前请求的偏好策略
	Strategy string `json:"strategy"` // "ttft_first", "tps_first", "success_first", "cost_first"
}

type Result struct {
	UpstreamModel string `json:"upstreamModel"`
	*models.Channel
}

type ResultRecord struct {
	ModelID      string `json:"modelId"`
	ChannelID    int64  `json:"channelId"`
	ChannelName  string `json:"channelName"`
	OutputTokens int64  `json:"outputTokens"`
	TTFT         int64  `json:"ttft"`     //ms
	Duration     int64  `json:"duration"` //ms
	Status       int    `json:"status"`   // 1: success, 2: fail
	StatusCode   int    `json:"statusCode"`
}
