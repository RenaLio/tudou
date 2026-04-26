package loadbalancer

import (
	"sync"

	"github.com/RenaLio/tudou/internal/models"
)

// Registry 存储数据
type Registry struct {
	mu        sync.RWMutex
	Channels  map[int64]*Channel
	Groups    map[int64]*Group
	Endpoints map[string]map[int64]*Endpoint // map[modelId]map[channelId]Endpoint
}

func NewRegistry() *Registry {
	return &Registry{
		Channels:  make(map[int64]*Channel),
		Groups:    make(map[int64]*Group),
		Endpoints: make(map[string]map[int64]*Endpoint),
	}
}

// ReloadChannel 加载或更新通道
func (r *Registry) ReloadChannel(channel *models.Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Channels[channel.ID] == nil {
		r.Channels[channel.ID] = &Channel{}
	}
	r.Channels[channel.ID].UpdateChannel(channel)
	channelModelMapping := channel.Models()
	r.removeChannelEndpointsLocked(channel.ID)
	for callModel, upstreamModel := range channelModelMapping {
		if r.Endpoints[callModel] == nil {
			r.Endpoints[callModel] = make(map[int64]*Endpoint)
		}
		r.Endpoints[callModel][channel.ID] = &Endpoint{
			ChannelID:      channel.ID,
			ChannelType:    string(channel.Type),
			Model:          callModel,
			UpstreamModel:  upstreamModel,
			BaseWeight:     int64(channel.Weight),
			CostRate:       channel.PriceRate,
			mu:             sync.RWMutex{},
			EmaTTFT:        1600,
			EmaTPS:         100,
			EmaSuccessRate: 1.0,
		}
	}
}

func (r *Registry) UnregisterChannel(channelId int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Channels, channelId)
	r.removeChannelEndpointsLocked(channelId)
}

func (r *Registry) GetChannelById(channelId int64) *Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Channels[channelId]
}

// ReloadGroup 加载或更新分组
func (r *Registry) ReloadGroup(group *models.ChannelGroup) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Groups[group.ID] == nil {
		r.Groups[group.ID] = &Group{}
	}
	r.Groups[group.ID].UpdateGroup(group)
}

func (r *Registry) UnregisterGroup(groupId int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Groups, groupId)
}

func (r *Registry) GetChannelsByGroupId(groupId int64) []*Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.Groups[groupId] == nil {
		return []*Channel{}
	}
	channelIds := r.Groups[groupId].Channels()
	channels := make([]*Channel, 0, len(channelIds))
	for _, channelId := range channelIds {
		if r.Channels[channelId] == nil {
			continue
		}
		channels = append(channels, r.Channels[channelId])
	}
	return channels
}

func (r *Registry) FilterAvailableChannel(channels []*Channel) []*Channel {
	data := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		if channel.IsAvailable() {
			data = append(data, channel)
		}
	}
	return data
}

func (r *Registry) FilterChannelByModel(model string, channels []*Channel) []*Channel {
	data := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		if channel.IsSupportModel(model) {
			data = append(data, channel)
		}
	}
	return data
}

func (r *Registry) GetEndpoint(model string, channelId int64) *Endpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Endpoints[model][channelId]
}

func (r *Registry) GetEndpoints(model string, channelIds ...int64) []*Endpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	endpoints := make([]*Endpoint, 0, len(channelIds))
	for _, channelId := range channelIds {
		if r.Endpoints[model][channelId] == nil {
			continue
		}
		endpoints = append(endpoints, r.Endpoints[model][channelId])
	}
	return endpoints
}

func (r *Registry) removeChannelEndpointsLocked(channelID int64) {
	for model, m := range r.Endpoints {
		delete(m, channelID)
		if len(m) == 0 {
			delete(r.Endpoints, model)
		}
	}
}

func (c *Registry) ExportRegistryData() Registry {
	return *c
}
