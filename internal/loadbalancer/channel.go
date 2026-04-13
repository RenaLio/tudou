package loadbalancer

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

// Channel 负载均衡器中的渠道包装
type Channel struct {
	*models.Channel
	ActiveConns   int64   // 当前活跃连接数
	LastUsedAt    int64   // 最后使用时间戳
	SuccessRate   float64 // 成功率 0-1.0，展示使用，不参与决策
	supportModels map[string]struct{}
	mu            sync.RWMutex
}

// IncConn 增加连接数
func (c *Channel) IncConn() {
	atomic.AddInt64(&c.ActiveConns, 1)
	atomic.StoreInt64(&c.LastUsedAt, time.Now().Unix())
}

// DecConn 减少连接数
func (c *Channel) DecConn() {
	atomic.AddInt64(&c.ActiveConns, -1)
}

func (c *Channel) IsAvailable() bool {
	connPass := true
	if c.Channel.Settings.MaxConcurrent > 0 {
		connPass = c.ActiveConns <= int64(c.Channel.Settings.MaxConcurrent)
	}
	return connPass && c.Channel.IsAvailable()
}

func (c *Channel) UpdateChannel(channel *models.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Channel = channel
	channelModels := channel.Models()
	c.supportModels = make(map[string]struct{}, len(channelModels))
	for model := range channelModels {
		c.supportModels[model] = struct{}{}
	}
}

func (c *Channel) UpdateSuccessRate(ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ok {
		c.SuccessRate = SuccessRate*c.SuccessRate + (1-SuccessRate)*float64(1)
	} else {
		c.SuccessRate = SuccessRate*c.SuccessRate + (1-SuccessRate)*float64(0)
	}
}

func (c *Channel) IsSupportModel(model string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.supportModels[model]
	return ok
}
