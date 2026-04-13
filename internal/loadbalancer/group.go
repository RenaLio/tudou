package loadbalancer

import (
	"sync"

	"github.com/RenaLio/tudou/internal/models"
)

type Group struct {
	*models.ChannelGroup

	mu sync.RWMutex
}

func (g *Group) UpdateGroup(group *models.ChannelGroup) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ChannelGroup = group
}

func (g *Group) GetGroup() *models.ChannelGroup {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.ChannelGroup
}

func (g *Group) Channels() []int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	data := make([]int64, 0, len(g.ChannelGroup.Channels))
	for _, channel := range g.ChannelGroup.Channels {
		data = append(data, channel.ID)
	}
	return data
}
