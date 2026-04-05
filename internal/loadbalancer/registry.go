package loadbalancer

import (
	"sync"

	"github.com/RenaLio/tudou/internal/models"
)

// Registry 存储数据
type Registry struct {
	mu       sync.RWMutex
	Channels map[int64]*Channel
	Groups   map[int64]*Group
	//Endpoints  map[string][]*Endpoint
	Endpoints map[string]map[string]*Endpoint // map[modelId]map[channelId]Endpoint
}

func NewRegistry() *Registry {
	return &Registry{
		Channels:  make(map[int64]*Channel),
		Groups:    make(map[int64]*Group),
		Endpoints: make(map[string]map[string]*Endpoint),
	}
}

func (r *Registry) UpdateChannel(channel *models.Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Channels[channel.ID] == nil {
		r.Channels[channel.ID] = &Channel{}
	}
	r.Channels[channel.ID].UpdateChannel(channel)
}

func (r *Registry) UpdateGroup(group *Group) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Groups[group.ID] = group
}
