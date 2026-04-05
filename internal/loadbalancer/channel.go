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
	ActiveConns int64 // 当前活跃连接数
	LastUsedAt  int64 // 最后使用时间戳

	SuccessRate float64 // 成功率 0-100.00

	mu sync.RWMutex
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
	return c.ActiveConns < int64(c.Channel.MaxConnections) && c.Channel.IsAvailable()
}

func (c *Channel) UpdateChannel(channel *models.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Channel = channel
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
