package relay

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

// ClientRegistry 按 channel.ID 缓存 *base.Client，感知 channel.UpdatedAt 变化自动失效。
type ClientRegistry struct {
	httpC *http.Client
	mu    sync.RWMutex
	m     map[int64]*cachedClient
}

type cachedClient struct {
	client    *base.Client
	updatedAt time.Time
}

// NewClientRegistry 构造 Registry；httpC 必须非 nil。
func NewClientRegistry(httpC *http.Client) *ClientRegistry {
	return &ClientRegistry{
		httpC: httpC,
		m:     make(map[int64]*cachedClient),
	}
}

// Get 获取或创建对应 channel 的 Client；当 channel.UpdatedAt 比缓存新时强制重建。
func (r *ClientRegistry) Get(ch *models.Channel, abilities []types.Ability) *base.Client {
	if ch == nil {
		return nil
	}
	r.mu.RLock()
	cached := r.m[ch.ID]
	r.mu.RUnlock()
	if cached != nil && !ch.UpdatedAt.After(cached.updatedAt) {
		return cached.client
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// double check
	if cached, ok := r.m[ch.ID]; ok && !ch.UpdatedAt.After(cached.updatedAt) {
		return cached.client
	}
	id := strconv.FormatInt(ch.ID, 10)
	client := base.NewClient(r.httpC, ch.BaseURL, ch.APIKey, id, abilities)
	r.m[ch.ID] = &cachedClient{client: client, updatedAt: ch.UpdatedAt}
	return client
}

// Invalidate 显式失效指定 channel 的缓存。
func (r *ClientRegistry) Invalidate(channelID int64) {
	r.mu.Lock()
	delete(r.m, channelID)
	r.mu.Unlock()
}
