package loadbalancer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"sync"

	"github.com/RenaLio/tudou/pkg/provider/plog"
)

type ScorePlugin func([]*Endpoint) []*Endpoint

type LoadBalancer interface {
	Select(ctx context.Context, req *Request, plugins ...ScorePlugin) ([]*Result, error)
}

type MetricsCollector interface {
	CollectMetrics(ctx context.Context, record *ResultRecord) error
	IncConn(channelID int64)
	DecConn(channelID int64)
}

type DynamicLoadBalancer struct {
	*Registry
}

func NewDynamicLoadBalancer(registry *Registry) *DynamicLoadBalancer {
	return &DynamicLoadBalancer{
		Registry: registry,
	}
}

func (lb *DynamicLoadBalancer) Select(ctx context.Context, req *Request, plugins ...ScorePlugin) ([]*Result, error) {
	channels := lb.GetChannelsByGroupId(req.GroupID)
	availableChannels := lb.FilterAvailableChannel(channels)
	modelChannels := lb.FilterChannelByModel(req.Model, availableChannels)
	if len(modelChannels) == 0 {
		return nil, ErrNoAvailableChannel
	}
	channelIds := make([]int64, 0, len(modelChannels))
	for _, channel := range modelChannels {
		channelIds = append(channelIds, channel.ID)
	}
	endpoints := lb.GetEndpoints(req.Model, channelIds...)
	if len(endpoints) == 0 {
		return nil, ErrNoAvailableChannel
	}
	plog.Debug("endpoints_step1", fmt.Sprintf("%#v", endpoints))
	endpoints = FilterAvailableEndpoints(endpoints)
	plog.Debug("endpoints_step2", fmt.Sprintf("%#v", endpoints))
	if len(endpoints) == 0 {
		return nil, ErrNoAvailableChannel
	}
	// sort
	sortedEndpoints := SortEndpoints(endpoints, req.Strategy, lb)
	plog.Debug("sorted_endpoints", fmt.Sprintf("%v", sortedEndpoints))
	// 随机扰动
	randNum := rand.IntN(101)
	plog.Debug("randNum", randNum)
	if randNum >= 90 {
		sortedEndpoints = Shuffled(sortedEndpoints)
		plog.Debug("shuffled_endpoints", fmt.Sprintf("%v", sortedEndpoints))
	}
	// 插件
	for _, plugin := range plugins {
		sortedEndpoints = plugin(sortedEndpoints)
	}

	result := make([]*Result, 0, len(sortedEndpoints))
	for _, endpoint := range sortedEndpoints {
		channel := lb.GetChannelById(endpoint.ChannelID)
		if channel == nil {
			continue
		}
		result = append(result, &Result{
			Channel:       *channel.Channel,
			UpstreamModel: endpoint.UpstreamModel,
		})
	}
	return result, nil
}

func FilterAvailableEndpoints(endpoints []*Endpoint) []*Endpoint {
	availableEndpoints := make([]*Endpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint == nil {
			continue
		}
		if endpoint.IsAvailable() {
			availableEndpoints = append(availableEndpoints, endpoint)
		}
	}
	return availableEndpoints
}

func SortEndpoints(endpoints []*Endpoint, strategy string, lb *DynamicLoadBalancer) []*Endpoint {
	scoreCache := make(map[int64]float64)

	switch strategy {
	case "random":
		return Shuffled(endpoints)
	case "performance":
		sort.Slice(endpoints, func(i, j int) bool {
			var scoreI, scoreJ float64
			if val, ok := scoreCache[endpoints[i].ChannelID]; ok {
				scoreI = val
			} else {
				scoreI = endpoints[i].ScoreWithWeights(DefaultPerformanceWeights)
				scoreCache[endpoints[i].ChannelID] = scoreI
			}
			if val, ok := scoreCache[endpoints[j].ChannelID]; ok {
				scoreJ = val
			} else {
				scoreJ = endpoints[j].ScoreWithWeights(DefaultPerformanceWeights)
				scoreCache[endpoints[j].ChannelID] = scoreJ
			}
			return scoreI > scoreJ
		})
	case "ttft_first":
		sort.Slice(endpoints, func(i, j int) bool {
			var scoreI, scoreJ float64
			if val, ok := scoreCache[endpoints[i].ChannelID]; ok {
				scoreI = val
			} else {
				scoreI = endpoints[i].ScoreWithWeights(TTFTFirstWeights)
				scoreCache[endpoints[i].ChannelID] = scoreI
			}
			if val, ok := scoreCache[endpoints[j].ChannelID]; ok {
				scoreJ = val
			} else {
				scoreJ = endpoints[j].ScoreWithWeights(TTFTFirstWeights)
				scoreCache[endpoints[j].ChannelID] = scoreJ
			}
			return scoreI > scoreJ
		})
	case "tps_first":
		sort.Slice(endpoints, func(i, j int) bool {
			return endpoints[i].ScoreWithWeights(TPSFirstWeights) > endpoints[j].ScoreWithWeights(TPSFirstWeights)
		})
	case "success_first":
		sort.Slice(endpoints, func(i, j int) bool {
			return endpoints[i].ScoreWithWeights(SuccessFirstWeights) > endpoints[j].ScoreWithWeights(SuccessFirstWeights)
		})
	case "cost_first":
		sort.Slice(endpoints, func(i, j int) bool {
			return endpoints[i].ScoreWithWeights(CostFirstWeights) > endpoints[j].ScoreWithWeights(CostFirstWeights)
		})
	case "weighted":
		scoreMap := make(map[int64]float64)
		for _, endpoint := range endpoints {
			score := endpoint.ScoreWithWeights(WeightedWeights)
			if score == 0.0 {
				score = 0.1
			}
			scoreMap[endpoint.ChannelID] = -(math.Log(rand.ExpFloat64()) / score)
		}
		sort.Slice(endpoints, func(i, j int) bool {
			if scoreMap[endpoints[i].ChannelID] > scoreMap[endpoints[j].ChannelID] {
				return true
			}
			return false
		})
	case "least_conn":
		sort.Slice(endpoints, func(i, j int) bool {
			channelI := endpoints[i].ChannelID
			channelJ := endpoints[j].ChannelID
			activeI, activeJ := 0, 0
			if data := lb.GetChannelById(channelI); data != nil {
				activeI = int(data.ActiveConns)
			} else {
				return false
			}
			if data := lb.GetChannelById(channelJ); data != nil {
				activeJ = int(data.ActiveConns)
			} else {
				return false
			}
			return activeI < activeJ
		})
	default:
		sort.Slice(endpoints, func(i, j int) bool {
			var scoreI, scoreJ float64
			if val, ok := scoreCache[endpoints[i].ChannelID]; ok {
				scoreI = val
			} else {
				scoreI = endpoints[i].ScoreWithWeights(DefaultPerformanceWeights)
				scoreCache[endpoints[i].ChannelID] = scoreI
			}
			if val, ok := scoreCache[endpoints[j].ChannelID]; ok {
				scoreJ = val
			} else {
				scoreJ = endpoints[j].ScoreWithWeights(DefaultPerformanceWeights)
				scoreCache[endpoints[j].ChannelID] = scoreJ
			}
			return scoreI > scoreJ
		})
	}
	return endpoints
}

func Shuffled[T any](src []T) []T {
	rand.Shuffle(len(src), func(i, j int) {
		src[i], src[j] = src[j], src[i]
	})
	return src
}

type AsyncMetricsCollector struct {
	*Registry
	eventCh chan *ResultRecord
	once    sync.Once
}

func NewAsyncMetricsCollector(reg *Registry, buffer int) *AsyncMetricsCollector {
	if buffer <= 0 {
		buffer = 1024
	}
	collector := &AsyncMetricsCollector{
		Registry: reg,
		eventCh:  make(chan *ResultRecord, buffer),
	}

	go collector.AsyncUpdateEndpoint()

	return collector
}

func (a *AsyncMetricsCollector) IncConn(channelID int64) {
	channel := a.GetChannelById(channelID)
	if channel != nil {
		channel.IncConn()
	}
}

func (a *AsyncMetricsCollector) DecConn(channelID int64) {
	channel := a.GetChannelById(channelID)
	if channel != nil {
		channel.DecConn()
	}
}

func (a *AsyncMetricsCollector) CollectMetrics(ctx context.Context, record *ResultRecord) error {
	a.DecConn(record.ChannelID)
	// update endpoint
	a.eventCh <- record
	return nil
}

func (a *AsyncMetricsCollector) AsyncUpdateEndpoint() {
	for data := range a.eventCh {
		// 人为的错误，跳过
		if data.StatusCode == 400 {
			continue
		}
		endpoint := a.GetEndpoint(data.Model, data.ChannelID)
		channel := a.GetChannelById(data.ChannelID)
		if endpoint != nil {
			isSuccess := data.Status == 1 && data.StatusCode == 200
			tps := 0.0
			if isSuccess {
				if data.Duration == 0 {
					data.Duration = 1
				}
				tps = float64(data.OutputTokens) * 1000 / (float64(data.Duration))
			}
			endpoint.UpdateMetrics(isSuccess, float64(data.TTFT), tps)
			if channel != nil {
				channel.UpdateSuccessRate(isSuccess)
			}
		}

	}
}

var (
	ErrNoAvailableChannel = errors.New("no available channel")
)

// PerformanceWeights 性能指标权重配置
type PerformanceWeights struct {
	SuccessRate float64 // 成功率权重
	TTFT        float64 // 首字时间权重
	TPS         float64 // 吞吐量权重
	weight      float64 // 基础权重
	Cost        float64 // 成本权重
}

// DefaultPerformanceWeights 默认权重配置
var DefaultPerformanceWeights = PerformanceWeights{
	SuccessRate: 0.25,
	TTFT:        0.35,
	TPS:         0.20,
	weight:      0.10,
	Cost:        0.10,
}

// TTFTFirstWeights TTFT优先权重
var TTFTFirstWeights = PerformanceWeights{
	SuccessRate: 0.20,
	TTFT:        0.50,
	TPS:         0.10,
	weight:      0.20,
}

// TPSFirstWeights TPS优先权重
var TPSFirstWeights = PerformanceWeights{
	SuccessRate: 0.20,
	TTFT:        0.20,
	TPS:         0.50,
	weight:      0.10,
}

// SuccessFirstWeights 成功率优先权重
var SuccessFirstWeights = PerformanceWeights{
	SuccessRate: 0.60,
	TTFT:        0.20,
	TPS:         0.10,
	weight:      0.10,
}

// CostFirstWeights 成本优先权重
var CostFirstWeights = PerformanceWeights{
	SuccessRate: 0.20,
	TTFT:        0.20,
	TPS:         0.10,
	weight:      0.00,
	Cost:        0.50,
}

// WeightedWeights 权重优先权重
var WeightedWeights = PerformanceWeights{
	SuccessRate: 0.10,
	TTFT:        0.10,
	TPS:         0.10,
	weight:      0.50,
}
