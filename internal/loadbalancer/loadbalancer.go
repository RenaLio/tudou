package loadbalancer

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type LoadBalancer interface {
	Select(ctx context.Context, req *Request) ([]*Result, error)
}

type MetricsCollector interface {
	CollectMetrics(ctx context.Context, record *ResultRecord) error
}

// Strategy 负载均衡策略标识
type Strategy string

const (
	StrategyRoundRobin  Strategy = "round_robin" // 轮询
	StrategyWeighted    Strategy = "weighted"    // 加权轮询
	StrategyLeastConn   Strategy = "least_conn"  // 最少连接
	StrategyRandom      Strategy = "random"      // 随机
	StrategyPriority    Strategy = "priority"    // 优先级
	StrategyPerformance Strategy = "performance" // 性能优先
)

var (
	ErrNoAvailableChannel = errors.New("no available channel")
	ErrInvalidStrategy    = errors.New("invalid load balance strategy")
)

// Selector 负载均衡选择器函数类型
// 从可用渠道中选择主渠道和候选渠道列表
// channels: 已过滤的可用渠道列表
// 返回: primary-主渠道, candidates-候选渠道列表(已排序), error-错误
type Selector func(channels []*Channel) (primary *Channel, candidates []*Channel, err error)

// ChannelMetrics 渠道性能指标
type ChannelMetrics struct {
	AvgTTFT         int64   // 平均首字时间 (ms)
	AvgLatency      int64   // 平均延迟 (ms)
	SuccessRate     float64 // 成功率 (0-1)
	TokensPerSecond float64 // 每秒处理token数
	RequestCount    int64   // 总请求数
	SuccessCount    int64   // 成功请求数
}

// UpdateMetrics 更新性能指标
func (c *Channel) UpdateMetrics(ttft, latency int64, success bool, tokens int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	m := &c.Metrics
	m.RequestCount++
	if success {
		m.SuccessCount++
	}

	// 更新成功率
	m.SuccessRate = float64(m.SuccessCount) / float64(m.RequestCount)

	// 更新平均TTFT (指数移动平均)
	if m.AvgTTFT == 0 {
		m.AvgTTFT = ttft
	} else {
		m.AvgTTFT = int64(0.7*float64(m.AvgTTFT) + 0.3*float64(ttft))
	}

	// 更新平均延迟
	if m.AvgLatency == 0 {
		m.AvgLatency = latency
	} else {
		m.AvgLatency = int64(0.7*float64(m.AvgLatency) + 0.3*float64(latency))
	}

	// 更新TPS (简单计算)
	if latency > 0 {
		currentTPS := float64(tokens) / (float64(latency) / 1000)
		if m.TokensPerSecond == 0 {
			m.TokensPerSecond = currentTPS
		} else {
			m.TokensPerSecond = 0.7*m.TokensPerSecond + 0.3*currentTPS
		}
	}
}

// PerformanceWeights 性能指标权重配置
type PerformanceWeights struct {
	SuccessRate float64 // 成功率权重
	TTFT        float64 // 首字时间权重
	TPS         float64 // 吞吐量权重
	Latency     float64 // 延迟权重
	Connections float64 // 活跃连接权重
}

// DefaultPerformanceWeights 默认权重配置
var DefaultPerformanceWeights = PerformanceWeights{
	SuccessRate: 0.40,
	TTFT:        0.20,
	TPS:         0.30,
	Latency:     0.00, // 默认不启用
	Connections: 0.10,
}

// TTFTFirstWeights TTFT优先权重
var TTFTFirstWeights = PerformanceWeights{
	SuccessRate: 0.20,
	TTFT:        0.50,
	TPS:         0.10,
	Latency:     0.10,
	Connections: 0.10,
}

// TPSFirstWeights TPS优先权重
var TPSFirstWeights = PerformanceWeights{
	SuccessRate: 0.20,
	TTFT:        0.10,
	TPS:         0.50,
	Latency:     0.10,
	Connections: 0.10,
}

// SuccessFirstWeights 成功率优先权重
var SuccessFirstWeights = PerformanceWeights{
	SuccessRate: 0.60,
	TTFT:        0.10,
	TPS:         0.10,
	Latency:     0.10,
	Connections: 0.10,
}

// LatencyFirstWeights 延迟优先权重
var LatencyFirstWeights = PerformanceWeights{
	SuccessRate: 0.20,
	TTFT:        0.10,
	TPS:         0.10,
	Latency:     0.50,
	Connections: 0.10,
}

// ScoreWithWeights 使用指定权重计算渠道得分
func (c *Channel) ScoreWithWeights(w PerformanceWeights) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := &c.Metrics
	if m.RequestCount == 0 {
		return float64(c.Weight) * 100
	}

	score := 0.0

	// 成功率得分
	if w.SuccessRate > 0 {
		score += m.SuccessRate * 100 * w.SuccessRate
	}

	// TTFT得分 (最佳100ms，超过1000ms得0分)
	if w.TTFT > 0 {
		ttftScore := 0.0
		if m.AvgTTFT < 100 {
			ttftScore = 100
		} else if m.AvgTTFT < 1000 {
			ttftScore = 100 - float64(m.AvgTTFT-100)/9
		}
		score += ttftScore * w.TTFT
	}

	// TPS得分 (最佳1000tps)
	if w.TPS > 0 {
		tpsScore := 0.0
		if m.TokensPerSecond > 1000 {
			tpsScore = 100
		} else {
			tpsScore = m.TokensPerSecond / 10
		}
		score += ttpsScore * w.TPS
	}

	// 延迟得分 (最佳100ms，超过1000ms得0分)
	if w.Latency > 0 {
		latencyScore := 0.0
		if m.AvgLatency < 100 {
			latencyScore = 100
		} else if m.AvgLatency < 1000 {
			latencyScore = 100 - float64(m.AvgLatency-100)/9
		}
		score += latencyScore * w.Latency
	}

	// 活跃连接得分 (最佳10个，超过100个得0分)
	if w.Connections > 0 {
		connScore := 0.0
		conns := atomic.LoadInt64(&c.ActiveConns)
		if conns < 10 {
			connScore = 100
		} else if conns < 100 {
			connScore = 100 - float64(conns-10)*1.11
		}
		score += connScore * w.Connections
	}

	return score
}

// Score 计算渠道综合得分 (使用默认权重)
func (c *Channel) Score() float64 {
	return c.ScoreWithWeights(DefaultPerformanceWeights)
}

// defaultSelectors 内置选择器函数
var defaultSelectors = map[Strategy]Selector{
	StrategyRoundRobin:  newRoundRobinSelector(),
	StrategyWeighted:    newWeightedSelector(),
	StrategyLeastConn:   newLeastConnSelector(),
	StrategyRandom:      newRandomSelector(),
	StrategyPriority:    newPrioritySelector(),
	StrategyPerformance: newPerformanceSelector(),
}

// newRoundRobinSelector 轮询选择器
func newRoundRobinSelector() Selector {
	var counter uint64
	return func(channels []*Channel) (primary *Channel, candidates []*Channel, err error) {
		if len(channels) == 0 {
			return nil, nil, ErrNoAvailableChannel
		}
		if len(channels) == 1 {
			return channels[0], nil, nil
		}
		idx := atomic.AddUint64(&counter, 1) % uint64(len(channels))
		primary = channels[idx]
		candidates = make([]*Channel, 0, len(channels)-1)
		for i, ch := range channels {
			if i != int(idx) {
				candidates = append(candidates, ch)
			}
		}
		return primary, candidates, nil
	}
}

// newWeightedSelector 加权选择器
func newWeightedSelector() Selector {
	return func(channels []*Channel) (primary *Channel, candidates []*Channel, err error) {
		if len(channels) == 0 {
			return nil, nil, ErrNoAvailableChannel
		}
		if len(channels) == 1 {
			return channels[0], nil, nil
		}
		// 按权重排序，权重高的优先
		sorted := make([]*Channel, len(channels))
		copy(sorted, channels)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Weight > sorted[j].Weight
		})
		return sorted[0], sorted[1:], nil
	}
}

// newLeastConnSelector 最少连接选择器
func newLeastConnSelector() Selector {
	return func(channels []*Channel) (primary *Channel, candidates []*Channel, err error) {
		if len(channels) == 0 {
			return nil, nil, ErrNoAvailableChannel
		}
		if len(channels) == 1 {
			return channels[0], nil, nil
		}
		// 按连接数排序，连接数少的优先
		sorted := make([]*Channel, len(channels))
		copy(sorted, channels)
		sort.Slice(sorted, func(i, j int) bool {
			return atomic.LoadInt64(&sorted[i].ActiveConns) < atomic.LoadInt64(&sorted[j].ActiveConns)
		})
		return sorted[0], sorted[1:], nil
	}
}

// newRandomSelector 随机选择器
func newRandomSelector() Selector {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var mu sync.Mutex
	return func(channels []*Channel) (primary *Channel, candidates []*Channel, err error) {
		if len(channels) == 0 {
			return nil, nil, ErrNoAvailableChannel
		}
		if len(channels) == 1 {
			return channels[0], nil, nil
		}
		mu.Lock()
		idx := rng.Intn(len(channels))
		mu.Unlock()
		primary = channels[idx]
		candidates = make([]*Channel, 0, len(channels)-1)
		for i, ch := range channels {
			if i != idx {
				candidates = append(candidates, ch)
			}
		}
		return primary, candidates, nil
	}
}

// newPrioritySelector 优先级选择器
func newPrioritySelector() Selector {
	return func(channels []*Channel) (primary *Channel, candidates []*Channel, err error) {
		if len(channels) == 0 {
			return nil, nil, ErrNoAvailableChannel
		}
		if len(channels) == 1 {
			return channels[0], nil, nil
		}
		// 按优先级排序，优先级高的优先
		sorted := make([]*Channel, len(channels))
		copy(sorted, channels)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Weight > sorted[j].Weight
		})
		return sorted[0], sorted[1:], nil
	}
}

// NewPerformanceSelector 创建性能优先选择器（可配置权重）
func NewPerformanceSelector(weights PerformanceWeights) Selector {
	return func(channels []*Channel) (primary *Channel, candidates []*Channel, err error) {
		if len(channels) == 0 {
			return nil, nil, ErrNoAvailableChannel
		}
		if len(channels) == 1 {
			return channels[0], nil, nil
		}

		// 按指定权重计算得分并排序
		sorted := make([]*Channel, len(channels))
		copy(sorted, channels)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].ScoreWithWeights(weights) > sorted[j].ScoreWithWeights(weights)
		})

		return sorted[0], sorted[1:], nil
	}
}

// newPerformanceSelector 性能优先选择器（使用默认权重）
func newPerformanceSelector() Selector {
	return NewPerformanceSelector(DefaultPerformanceWeights)
}

// Factory 负载均衡器工厂
type Factory struct {
	selectors map[Strategy]Selector
	mu        sync.RWMutex
}

// NewFactory 创建负载均衡器工厂
func NewFactory() *Factory {
	selectors := make(map[Strategy]Selector)
	for k, v := range defaultSelectors {
		selectors[k] = v
	}
	return &Factory{
		selectors: selectors,
	}
}

// Get 获取选择器
func (f *Factory) Get(strategy Strategy) (Selector, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	selector, ok := f.selectors[strategy]
	if !ok {
		return nil, ErrInvalidStrategy
	}
	return selector, nil
}

// Register 注册自定义选择器
func (f *Factory) Register(strategy Strategy, selector Selector) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.selectors[strategy] = selector
}

// MustRegister 强制注册或替换选择器（用于内置策略覆盖）
func (f *Factory) MustRegister(strategy Strategy, selector Selector) {
	f.Register(strategy, selector)
}

// filterAvailable 过滤可用渠道
func filterAvailable(channels []*Channel) []*Channel {
	available := make([]*Channel, 0, len(channels))
	for _, ch := range channels {
		if ch.IsAvailable() {
			available = append(available, ch)
		}
	}
	return available
}

// Manager 负载均衡管理器
type Manager struct {
	factory *Factory
	groups  sync.Map // map[int64]*GroupState
}

// GroupState 渠道组状态
type GroupState struct {
	GroupID      int64
	Channels     []*Channel
	ModelChannel map[int64][]*Channel // modelID -> channels
	Selector     Selector             // 选择器函数
	mu           sync.RWMutex
}

// SelectResult 选择结果
type SelectResult struct {
	Primary    *Channel   // 主渠道
	Candidates []*Channel // 候选渠道列表
	GroupID    int64      // 渠道组ID
}

// NewManager 创建负载均衡管理器
func NewManager() *Manager {
	return &Manager{
		factory: NewFactory(),
	}
}

// RegisterGroup 注册渠道组
func (m *Manager) RegisterGroup(group *models.ChannelGroup, channels []*models.Channel, modelMap map[int64][]int64) error {
	selector, err := m.factory.Get(Strategy(group.LoadBalanceStrategy))
	if err != nil {
		return err
	}

	wrapped := make([]*Channel, len(channels))
	channelMap := make(map[int64]*Channel)
	for i, ch := range channels {
		wrapped[i] = &Channel{Channel: ch}
		channelMap[ch.ID] = wrapped[i]
	}

	// 构建 model -> channels 映射
	modelChannel := make(map[int64][]*Channel)
	for modelID, channelIDs := range modelMap {
		for _, cid := range channelIDs {
			if ch, ok := channelMap[cid]; ok {
				modelChannel[modelID] = append(modelChannel[modelID], ch)
			}
		}
	}

	state := &GroupState{
		GroupID:      group.ID,
		Channels:     wrapped,
		ModelChannel: modelChannel,
		Selector:     selector,
	}

	m.groups.Store(group.ID, state)
	return nil
}

// RegisterGroupWithSelector 使用自定义选择器注册渠道组
func (m *Manager) RegisterGroupWithSelector(groupID int64, channels []*models.Channel, modelMap map[int64][]int64, selector Selector) {
	wrapped := make([]*Channel, len(channels))
	channelMap := make(map[int64]*Channel)
	for i, ch := range channels {
		wrapped[i] = &Channel{Channel: ch}
		channelMap[ch.ID] = wrapped[i]
	}

	// 构建 model -> channels 映射
	modelChannel := make(map[int64][]*Channel)
	for modelID, channelIDs := range modelMap {
		for _, cid := range channelIDs {
			if ch, ok := channelMap[cid]; ok {
				modelChannel[modelID] = append(modelChannel[modelID], ch)
			}
		}
	}

	state := &GroupState{
		GroupID:      groupID,
		Channels:     wrapped,
		ModelChannel: modelChannel,
		Selector:     selector,
	}

	m.groups.Store(groupID, state)
}

// Select 为指定渠道组选择渠道
// modelID: 模型ID，用于筛选支持该模型的渠道
func (m *Manager) Select(ctx context.Context, groupID int64, modelID int64) (*SelectResult, error) {
	value, ok := m.groups.Load(groupID)
	if !ok {
		return nil, errors.New("group not found")
	}

	state := value.(*GroupState)
	state.mu.RLock()
	defer state.mu.RUnlock()

	// 获取支持该模型的渠道列表
	channels, ok := state.ModelChannel[modelID]
	if !ok || len(channels) == 0 {
		return nil, errors.New("no channel supports this model")
	}

	// 过滤可用渠道
	available := filterAvailable(channels)
	if len(available) == 0 {
		return nil, ErrNoAvailableChannel
	}

	// 使用选择器函数选择渠道
	primary, candidates, err := state.Selector(available)
	if err != nil {
		return nil, err
	}

	return &SelectResult{
		Primary:    primary,
		Candidates: candidates,
		GroupID:    groupID,
	}, nil
}

// UpdateSelector 更新渠道组选择器
func (m *Manager) UpdateSelector(groupID int64, selector Selector) error {
	value, ok := m.groups.Load(groupID)
	if !ok {
		return errors.New("group not found")
	}

	state := value.(*GroupState)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.Selector = selector
	return nil
}

// UpdateChannels 更新渠道组渠道列表
func (m *Manager) UpdateChannels(groupID int64, channels []*models.Channel, modelMap map[int64][]int64) {
	value, ok := m.groups.Load(groupID)
	if !ok {
		return
	}

	state := value.(*GroupState)
	state.mu.Lock()
	defer state.mu.Unlock()

	wrapped := make([]*Channel, len(channels))
	channelMap := make(map[int64]*Channel)
	for i, ch := range channels {
		wrapped[i] = &Channel{Channel: ch}
		channelMap[ch.ID] = wrapped[i]
	}
	state.Channels = wrapped

	// 更新 model -> channels 映射
	modelChannel := make(map[int64][]*Channel)
	for modelID, channelIDs := range modelMap {
		for _, cid := range channelIDs {
			if ch, ok := channelMap[cid]; ok {
				modelChannel[modelID] = append(modelChannel[modelID], ch)
			}
		}
	}
	state.ModelChannel = modelChannel
}

// UnregisterGroup 注销渠道组
func (m *Manager) UnregisterGroup(groupID int64) {
	m.groups.Delete(groupID)
}
