package tasks

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/pkg/sid"
	"github.com/RenaLio/tudou/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// statsCounter 是一个临时的统计累加器，用于在内存中聚合请求日志的指标数据。
type statsCounter struct {
	inputToken                int64 // 输入 token 总量
	outputToken               int64 // 输出 token 总量
	cachedCreationInputTokens int64 // 缓存创建时涉及的输入 token 量
	cachedReadInputTokens     int64 // 缓存读取时涉及的输入 token 量
	requestSuccess            int64 // 成功请求次数
	requestFailed             int64 // 失败请求次数
	totalCostMicros           int64 // 总费用（微单位，例如微美元）
	ttftSum                   int64 // 首 token 时间（TTFT）的总和
	ttftCount                 int64 // 有效 TTFT 的计数，用于计算平均值
	transferTimeSum           int64 // 传输耗时的总和（毫秒）
}

// add 方法将一条请求日志的指标累加到当前计数器上。
func (c *statsCounter) add(log *models.RequestLog) {
	// 如果日志为空，直接返回，不做处理
	if log == nil {
		return
	}
	// 累加各项基础指标
	c.inputToken += log.InputToken
	c.outputToken += log.OutputToken
	c.cachedCreationInputTokens += log.CachedCreationInputTokens
	c.cachedReadInputTokens += log.CachedReadInputTokens
	c.totalCostMicros += log.CostMicros
	// 传输时间仅在有值时累加
	if log.TransferTime > 0 {
		c.transferTimeSum += log.TransferTime
	}
	// TTFT 仅在有值时累加，并增加计数
	if log.TTFT > 0 {
		c.ttftSum += log.TTFT
		c.ttftCount++
	}
	// 按日志状态统计成功与失败次数
	switch log.Status {
	case models.RequestStatusSuccess:
		c.requestSuccess++
	case models.RequestStatusFail:
		c.requestFailed++
	default:
		// 未知状态也视为失败
		c.requestFailed++
	}
}

// fillCommon 将累加器的通用基础指标填充到传入的字段指针中。
// 这用于将聚合结果写入最终的统计模型。
func (c *statsCounter) fillCommon(
	input *int64, // 输入 token 字段指针
	output *int64, // 输出 token 字段指针
	cachedCreation *int64, // 缓存创建输入 token 字段指针
	cachedRead *int64, // 缓存读取输入 token 字段指针
	success *int64, // 成功请求数字段指针
	failed *int64, // 失败请求数字段指针
	totalCost *int64, // 总费用字段指针
) {
	*input = c.inputToken
	*output = c.outputToken
	*cachedCreation = c.cachedCreationInputTokens
	*cachedRead = c.cachedReadInputTokens
	*success = c.requestSuccess
	*failed = c.requestFailed
	*totalCost = c.totalCostMicros
}

// avgTTFT 计算平均首 token 时间（毫秒），若没有有效记录则返回 0。
func (c *statsCounter) avgTTFT() int {
	if c.ttftCount <= 0 {
		return 0
	}
	// 总和除以计数得到整数平均值
	return int(c.ttftSum / c.ttftCount)
}

// avgTPS 计算平均输出 token 速率（每秒 token 数），不附加额外开销。
func (c *statsCounter) avgTPS() float64 {
	return c.avgTPSWithSuccessOverhead(0)
}

// avgTPSWithSuccessOverhead 计算平均 TPS，可对每个成功请求附加固定的额外开销（毫秒）。
// 这用于修正渠道级别 TPS 估算，以防止跨模型时过高估计。
func (c *statsCounter) avgTPSWithSuccessOverhead(overheadPerSuccessMs int64) float64 {
	if c.transferTimeSum <= 0 {
		return 0
	}
	// 分母 = 传输总耗时
	denominator := c.transferTimeSum
	// 如果指定了每次成功请求的额外开销，且存在成功请求，则将其加入分母
	if overheadPerSuccessMs > 0 && c.requestSuccess > 0 {
		denominator += c.requestSuccess * overheadPerSuccessMs
	}
	if denominator <= 0 {
		return 0
	}
	// TPS = 输出 token 数 * 1000 / 总耗时（毫秒）
	return float64(c.outputToken) * 1000 / float64(denominator)
}

// channelModelKey 用作映射键，组合渠道 ID 与模型名称。
type channelModelKey struct {
	channelID int64
	model     string
}

// dailyKey 用作映射键，组合用户 ID 与日期字符串。
type dailyKey struct {
	userID int64
	date   string
}

// hourlyKey 用作映射键，组合用户 ID、日期与小时数字。
type hourlyKey struct {
	userID int64
	date   string
	hour   int
}

// aggregationSnapshot 存储一次聚合操作生成的所有统计数据的快照。
type aggregationSnapshot struct {
	ChannelStats         []*models.ChannelStats         // 渠道统计列表
	ChannelModelStats    []*models.ChannelModelStats    // 渠道-模型统计列表
	TokenStats           []*models.TokenStats           // Token 统计列表
	UserStats            []*models.UserStats            // 用户统计列表
	UserUsageDailyStats  []*models.UserUsageDailyStats  // 用户每日用量统计列表
	UserUsageHourlyStats []*models.UserUsageHourlyStats // 用户每小时用量统计列表

	channelCounterMap      map[int64]*statsCounter           // 渠道 ID -> 计数器（用于后续合并计算）
	channelModelCounterMap map[channelModelKey]*statsCounter // 渠道模型键 -> 计数器
}

// aggregateRequestLogs 从一批请求日志中聚合生成所有维度的统计快照。
// nextID 是一个函数，用于生成新记录的唯一 ID。
func aggregateRequestLogs(logs []*models.RequestLog, nextID func() int64) *aggregationSnapshot {
	// 初始化各类存储 map，用于存放最终统计记录和对应的累加器
	channelStatsMap := make(map[int64]*models.ChannelStats)
	channelCounterMap := make(map[int64]*statsCounter)

	channelModelStatsMap := make(map[channelModelKey]*models.ChannelModelStats)
	channelModelCounterMap := make(map[channelModelKey]*statsCounter)

	tokenStatsMap := make(map[int64]*models.TokenStats)
	tokenCounterMap := make(map[int64]*statsCounter)

	userStatsMap := make(map[int64]*models.UserStats)
	userCounterMap := make(map[int64]*statsCounter)

	dailyStatsMap := make(map[dailyKey]*models.UserUsageDailyStats)
	dailyCounterMap := make(map[dailyKey]*statsCounter)

	hourlyStatsMap := make(map[hourlyKey]*models.UserUsageHourlyStats)
	hourlyCounterMap := make(map[hourlyKey]*statsCounter)

	// 遍历所有日志记录
	for _, item := range logs {
		if item == nil {
			continue // 跳过空日志
		}

		// 按渠道维度聚合
		if item.ChannelID > 0 {
			channelName := strings.TrimSpace(item.ChannelName)
			// 如果渠道记录不存在，则创建并关联计数器
			if _, exists := channelStatsMap[item.ChannelID]; !exists {
				channelStatsMap[item.ChannelID] = &models.ChannelStats{
					ChannelID:   item.ChannelID,
					ChannelName: channelName,
				}
				channelCounterMap[item.ChannelID] = &statsCounter{}
			} else if channelName != "" && channelStatsMap[item.ChannelID].ChannelName == "" {
				channelStatsMap[item.ChannelID].ChannelName = channelName
			}
			// 将当前日志累加到对应计数器
			channelCounterMap[item.ChannelID].add(item)
		}

		// 按渠道-模型维度聚合
		model := strings.TrimSpace(item.UpstreamModel) // 去除模型名称首尾空格
		if item.ChannelID > 0 && model != "" {
			key := channelModelKey{
				channelID: item.ChannelID,
				model:     model,
			}
			// 如果记录不存在则创建
			if _, exists := channelModelStatsMap[key]; !exists {
				channelModelStatsMap[key] = &models.ChannelModelStats{
					ChannelID: item.ChannelID,
					Model:     model,
				}
				channelModelCounterMap[key] = &statsCounter{}
			}
			channelModelCounterMap[key].add(item)
		}

		// 按 Token 维度聚合
		if item.TokenID > 0 {
			if _, exists := tokenStatsMap[item.TokenID]; !exists {
				tokenStatsMap[item.TokenID] = &models.TokenStats{
					TokenID: item.TokenID,
				}
				tokenCounterMap[item.TokenID] = &statsCounter{}
			}
			tokenCounterMap[item.TokenID].add(item)
		}

		// 按用户维度聚合（总计）
		if item.UserID > 0 {
			if _, exists := userStatsMap[item.UserID]; !exists {
				userStatsMap[item.UserID] = &models.UserStats{
					UserID: item.UserID,
				}
				userCounterMap[item.UserID] = &statsCounter{}
			}
			userCounterMap[item.UserID].add(item)
		}

		// 按用户每日、每小时维度聚合
		if item.UserID > 0 {
			createdAt := item.CreatedAt            // 使用记录时间
			date := createdAt.Format("2006-01-02") // 格式化为日期字符串 yyyy-MM-dd
			hour := createdAt.Hour()               // 获取一天中的小时数(0-23)

			// 按日期维度
			dKey := dailyKey{
				userID: item.UserID,
				date:   date,
			}
			if _, exists := dailyStatsMap[dKey]; !exists {
				dailyStatsMap[dKey] = &models.UserUsageDailyStats{
					ID:     nextIDOrZero(nextID), // 生成记录 ID
					UserID: item.UserID,
					Date:   date,
				}
				dailyCounterMap[dKey] = &statsCounter{}
			}
			dailyCounterMap[dKey].add(item)

			// 按小时维度
			hKey := hourlyKey{
				userID: item.UserID,
				date:   date,
				hour:   hour,
			}
			if _, exists := hourlyStatsMap[hKey]; !exists {
				hourlyStatsMap[hKey] = &models.UserUsageHourlyStats{
					ID:     nextIDOrZero(nextID), // 生成记录 ID
					UserID: item.UserID,
					Date:   date,
					Hour:   hour,
				}
				hourlyCounterMap[hKey] = &statsCounter{}
			}
			hourlyCounterMap[hKey].add(item)
		}
	}

	// 将聚合结果从计数器填充到最终的统计对象中
	// 渠道统计：填充基础字段并计算 AvgTTFT / AvgTPS
	for id, stats := range channelStatsMap {
		channelCounterMap[id].fillCommon(
			&stats.InputToken,
			&stats.OutputToken,
			&stats.CachedCreationInputTokens,
			&stats.CachedReadInputTokens,
			&stats.RequestSuccess,
			&stats.RequestFailed,
			&stats.TotalCostMicros,
		)
		stats.AvgTTFT = channelCounterMap[id].avgTTFT()
		// 渠道级吞吐按“传输耗时 + 成功请求固定调度开销(1s/次)”计算，避免跨模型聚合时过高估计
		stats.AvgTPS = channelCounterMap[id].avgTPSWithSuccessOverhead(1000)
	}

	// 渠道-模型统计
	for key, stats := range channelModelStatsMap {
		channelModelCounterMap[key].fillCommon(
			&stats.InputToken,
			&stats.OutputToken,
			&stats.CachedCreationInputTokens,
			&stats.CachedReadInputTokens,
			&stats.RequestSuccess,
			&stats.RequestFailed,
			&stats.TotalCostMicros,
		)
		stats.AvgTTFT = channelModelCounterMap[key].avgTTFT()
		stats.AvgTPS = channelModelCounterMap[key].avgTPS()
	}

	// Token 统计（没有 AvgTTFT 和 AvgTPS）
	for id, stats := range tokenStatsMap {
		tokenCounterMap[id].fillCommon(
			&stats.InputToken,
			&stats.OutputToken,
			&stats.CachedCreationInputTokens,
			&stats.CachedReadInputTokens,
			&stats.RequestSuccess,
			&stats.RequestFailed,
			&stats.TotalCostMicros,
		)
	}

	// 用户总计统计
	for id, stats := range userStatsMap {
		userCounterMap[id].fillCommon(
			&stats.InputToken,
			&stats.OutputToken,
			&stats.CachedCreationInputTokens,
			&stats.CachedReadInputTokens,
			&stats.RequestSuccess,
			&stats.RequestFailed,
			&stats.TotalCostMicros,
		)
	}

	// 用户每日统计
	for key, stats := range dailyStatsMap {
		dailyCounterMap[key].fillCommon(
			&stats.InputToken,
			&stats.OutputToken,
			&stats.CachedCreationInputTokens,
			&stats.CachedReadInputTokens,
			&stats.RequestSuccess,
			&stats.RequestFailed,
			&stats.TotalCostMicros,
		)
	}

	// 用户每小时统计
	for key, stats := range hourlyStatsMap {
		hourlyCounterMap[key].fillCommon(
			&stats.InputToken,
			&stats.OutputToken,
			&stats.CachedCreationInputTokens,
			&stats.CachedReadInputTokens,
			&stats.RequestSuccess,
			&stats.RequestFailed,
			&stats.TotalCostMicros,
		)
	}

	// 返回聚合快照，包含各类统计切片和计数器映射（用于后续合并）
	return &aggregationSnapshot{
		ChannelStats:           mapValues(channelStatsMap),
		ChannelModelStats:      mapValues(channelModelStatsMap),
		TokenStats:             mapValues(tokenStatsMap),
		UserStats:              mapValues(userStatsMap),
		UserUsageDailyStats:    mapValues(dailyStatsMap),
		UserUsageHourlyStats:   mapValues(hourlyStatsMap),
		channelCounterMap:      channelCounterMap,
		channelModelCounterMap: channelModelCounterMap,
	}
}

// nextIDOrZero 安全调用 nextID 函数，若 nextID 为 nil 则返回 0。
func nextIDOrZero(nextID func() int64) int64 {
	if nextID == nil {
		return 0
	}
	return nextID()
}

// mapValues 泛型函数，将 map 的值提取为切片，不保证顺序。
func mapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, value := range m {
		values = append(values, value)
	}
	return values
}

// StatsAggregationTaskStats 记录统计聚合任务的运行时状态信息。
type StatsAggregationTaskStats struct {
	LastRunAt             *time.Time    `json:"lastRunAt,omitempty"`   // 上次运行时间（指针，可为空）
	LastDuration          time.Duration `json:"lastDuration"`          // 上次运行耗时
	LastError             string        `json:"lastError,omitempty"`   // 上次错误信息
	ProcessedLogs         int           `json:"processedLogs"`         // 处理的日志数量
	ChannelStats          int           `json:"channelStats"`          // 生成的渠道统计记录数
	ChannelModelStats     int           `json:"channelModelStats"`     // 渠道-模型统计数
	TokenStats            int           `json:"tokenStats"`            // Token 统计数
	UserStats             int           `json:"userStats"`             // 用户统计数
	UserUsageDailyStats   int           `json:"userUsageDailyStats"`   // 用户每日统计数
	UserUsageHourlyStats  int           `json:"userUsageHourlyStats"`  // 用户每小时统计数
	LastObservationWindow string        `json:"lastObservationWindow"` // 最后一次处理的观测窗口（文本表示）
	LastStartID           int64         `json:"lastStartID,string"`    // 处理的起始日志 ID
	LastEndID             int64         `json:"lastEndID,string"`      // 处理的结束日志 ID
	LastTaskID            int64         `json:"lastTaskID,string"`     // 最近的任务记录 ID
	UpdatedAt             *time.Time    `json:"updatedAt,omitempty"`   // 状态更新时间
}

// StatsAggregationTask 是统计聚合任务的核心结构，包含所需依赖和运行时状态。
type StatsAggregationTask struct {
	logger                   *log.Logger                         // 日志记录器
	tm                       repository.Transaction              // 事务管理器
	aggregationTaskRepo      repository.AggregationTaskRepo      // 聚合任务记录仓库
	requestLogRepo           repository.RequestLogRepo           // 请求日志仓库
	channelStatsRepo         repository.ChannelStatsRepo         // 渠道统计仓库
	channelModelStatsRepo    repository.ChannelModelStatsRepo    // 渠道-模型统计仓库
	tokenStatsRepo           repository.TokenStatsRepo           // Token 统计仓库
	userStatsRepo            repository.UserStatsRepo            // 用户统计仓库
	userUsageDailyStatsRepo  repository.UserUsageDailyStatsRepo  // 用户每日统计仓库
	userUsageHourlyStatsRepo repository.UserUsageHourlyStatsRepo // 用户每小时统计仓库
	nextID                   func() int64                        // ID 生成函数
	mu                       sync.RWMutex                        // 读写锁，保护运行时状态
	stats                    StatsAggregationTaskStats           // 运行时状态
}

func NewStatsAggregationTask(
	logger *log.Logger,
	sid *sid.Sid,
	tm repository.Transaction,
	aggregationTaskRepo repository.AggregationTaskRepo,
	requestLogRepo repository.RequestLogRepo,
	channelStatsRepo repository.ChannelStatsRepo,
	channelModelStatsRepo repository.ChannelModelStatsRepo,
	tokenStatsRepo repository.TokenStatsRepo,
	userStatsRepo repository.UserStatsRepo,
	userUsageDailyStatsRepo repository.UserUsageDailyStatsRepo,
	userUsageHourlyStatsRepo repository.UserUsageHourlyStatsRepo,
) *StatsAggregationTask {
	return &StatsAggregationTask{
		logger:                   logger,
		tm:                       tm,
		aggregationTaskRepo:      aggregationTaskRepo,
		requestLogRepo:           requestLogRepo,
		channelStatsRepo:         channelStatsRepo,
		channelModelStatsRepo:    channelModelStatsRepo,
		tokenStatsRepo:           tokenStatsRepo,
		userStatsRepo:            userStatsRepo,
		userUsageDailyStatsRepo:  userUsageDailyStatsRepo,
		userUsageHourlyStatsRepo: userUsageHourlyStatsRepo,
		nextID:                   sidIDGenerator(sid),
	}
}

func sidIDGenerator(s *sid.Sid) func() int64 {
	if s == nil {
		return func() int64 {
			return time.Now().UnixNano()
		}
	}
	return s.GenInt64
}

func (t *StatsAggregationTask) Name() string {
	return StatsAggregationTaskName
}

// Run 执行一次聚合任务的主逻辑。
func (t *StatsAggregationTask) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background() // 若未传入 context，使用默认背景 context
	}
	startedAt := time.Now() // 记录任务开始时间
	now := startedAt

	// 获取上次成功完成的聚合任务的最大 EndID，作为增量拉取的起点
	lastEndID, err := t.getLastCompletedEndID(ctx)
	if err != nil {
		t.updateRuntimeState(startedAt, now, nil, 0, 0, 0, 0, err)
		return err
	}

	// 拉取增量请求日志
	logs, err := t.loadIncrementalRequestLogs(ctx, lastEndID)
	if err != nil {
		t.updateRuntimeState(startedAt, now, nil, 0, lastEndID, lastEndID, 0, err)
		return err
	}
	// 如果没有新日志，更新状态并返回
	if len(logs) == 0 {
		refreshSnapshot, refreshErr := t.buildWindowRefreshSnapshot(ctx)
		if refreshErr != nil {
			t.updateRuntimeState(startedAt, now, nil, 0, lastEndID, lastEndID, 0, refreshErr)
			return refreshErr
		}
		if refreshSnapshot != nil && (len(refreshSnapshot.ChannelStats) > 0 || len(refreshSnapshot.ChannelModelStats) > 0) {
			refreshErr = t.refreshObservationWindowsOnly(ctx, refreshSnapshot, now)
			t.updateRuntimeState(startedAt, now, refreshSnapshot, 0, lastEndID, lastEndID, 0, refreshErr)
			return refreshErr
		}
		t.updateRuntimeState(startedAt, now, nil, 0, lastEndID, lastEndID, 0, nil)
		return nil
	}

	// 本批日志的 ID 范围
	startID := logs[0].ID
	endID := logs[len(logs)-1].ID

	// 创建聚合任务记录，状态设为运行中
	taskRecord := &models.AggregationTask{
		ID:       t.nextID(),
		TaskName: t.Name(),
		StartID:  startID,
		EndID:    endID,
		Status:   int8(models.AggregationTaskStatusRunning),
	}
	// 如果 ID 生成返回 0（如 sid 未初始化），则使用时间戳作为后备
	if taskRecord.ID <= 0 {
		taskRecord.ID = time.Now().UnixNano()
	}
	// 检查仓库依赖
	if t.aggregationTaskRepo == nil {
		err = errors.New("aggregation task repo is nil")
		t.updateRuntimeState(startedAt, now, nil, len(logs), startID, endID, taskRecord.ID, err)
		return err
	}
	// 持久化任务记录
	if err = t.aggregationTaskRepo.Create(ctx, taskRecord); err != nil {
		t.updateRuntimeState(startedAt, now, nil, len(logs), startID, endID, taskRecord.ID, err)
		return err
	}

	// 扁平化日志
	logs = logsNormalize(logs)
	// 聚合日志生成快照
	snapshot := aggregateRequestLogs(logs, t.nextID)
	// 合并快照并持久化所有统计数据
	err = t.mergeAndPersistSnapshot(ctx, snapshot, now, taskRecord)
	// 更新运行时状态（无论成功或失败）
	t.updateRuntimeState(startedAt, now, snapshot, len(logs), startID, endID, taskRecord.ID, err)
	return err
}

// CurrentStats 返回当前聚合任务的运行时状态副本，线程安全。
func (t *StatsAggregationTask) CurrentStats() (any, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// 深拷贝时间指针以避免外部修改
	cloned := t.stats
	cloned.LastRunAt = cloneTimePtr(t.stats.LastRunAt)
	cloned.UpdatedAt = cloneTimePtr(t.stats.UpdatedAt)
	return cloned, nil
}

// getLastCompletedEndID 查询上一次成功完成的聚合任务的 EndID，用于确定增量起点。
func (t *StatsAggregationTask) getLastCompletedEndID(ctx context.Context) (int64, error) {
	if t.aggregationTaskRepo == nil {
		return 0, errors.New("aggregation task repo is nil")
	}
	// 获取最近的已完成任务记录
	task, err := t.aggregationTaskRepo.GetLatestCompletedByTaskName(ctx, t.Name())
	if err == nil {
		return task.EndID, nil
	}
	// 如果没找到记录，认为是从头开始，返回 0
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return 0, err
}

// loadIncrementalRequestLogs 从上次结束 ID 开始，分批加载所有增量请求日志。
func (t *StatsAggregationTask) loadIncrementalRequestLogs(ctx context.Context, afterID int64) ([]*models.RequestLog, error) {
	if t.requestLogRepo == nil {
		return nil, errors.New("request log repo is nil")
	}

	// 使用配置的最大分页大小，兜底默认200
	pageSize := repository.GetMaxPageSize()
	if pageSize <= 0 {
		pageSize = 200
	}

	cursor := afterID
	out := make([]*models.RequestLog, 0, pageSize) // 预分配空间
	for {
		// 按 ID 升序分页拉取
		items, _, err := t.requestLogRepo.List(ctx, repository.RequestLogListOption{
			Page:     1,
			PageSize: pageSize,
			OrderBy:  "id ASC",
			IDGT:     cursor, // 只取大于游标的记录
		})
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break // 没有更多记录
		}
		out = append(out, items...)
		cursor = items[len(items)-1].ID // 更新游标
	}
	return out, nil
}

// mergeAndPersistSnapshot 在事务中合并快照数据并持久化，同时更新任务状态。
func (t *StatsAggregationTask) mergeAndPersistSnapshot(
	ctx context.Context,
	snapshot *aggregationSnapshot,
	now time.Time,
	taskRecord *models.AggregationTask,
) error {
	if snapshot == nil {
		return nil
	}
	if t.tm == nil {
		return errors.New("transaction manager is nil")
	}

	// 在事务中执行合并与持久化
	txErr := t.tm.Transaction(ctx, func(txCtx context.Context) error {
		// 1. 处理观测窗口相关数据
		if err := t.applyObservationWindows(txCtx, snapshot, now); err != nil {
			return err
		}
		// 2. 持久化各类合并后的统计
		if err := t.persistMergedStats(txCtx, snapshot); err != nil {
			return err
		}
		// 3. 更新任务记录为已完成
		finishedAt := time.Now()
		taskRecord.Status = int8(models.AggregationTaskStatusDone)
		taskRecord.ErrorMsg = ""
		taskRecord.FinishedAt = &finishedAt
		return t.aggregationTaskRepo.Update(txCtx, taskRecord)
	})
	if txErr == nil {
		return nil
	}

	// 事务失败，记录失败状态
	failedAt := time.Now()
	taskRecord.Status = int8(models.AggregationTaskStatusFailed)
	taskRecord.RetryCount++             // 增加重试次数
	taskRecord.ErrorMsg = txErr.Error() // 记录错误信息
	taskRecord.FinishedAt = &failedAt
	_ = t.aggregationTaskRepo.Update(context.Background(), taskRecord) // 非事务更新，防止影响主事务
	return txErr
}

// buildWindowRefreshSnapshot 基于现有统计记录构造一个“仅刷新窗口”的快照。
// 该快照不包含增量计数，仅携带渠道和渠道-模型主键，用于后续重算并覆盖 Window3H。
func (t *StatsAggregationTask) buildWindowRefreshSnapshot(ctx context.Context) (*aggregationSnapshot, error) {
	if t.channelStatsRepo == nil {
		return nil, errors.New("channel stats repo is nil")
	}
	if t.channelModelStatsRepo == nil {
		return nil, errors.New("channel model stats repo is nil")
	}

	channelStats, err := t.channelStatsRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	snapshot := &aggregationSnapshot{
		ChannelStats:      make([]*models.ChannelStats, 0, len(channelStats)),
		ChannelModelStats: make([]*models.ChannelModelStats, 0, len(channelStats)),
	}

	seenModelKeys := make(map[channelModelKey]struct{})
	for _, item := range channelStats {
		if item == nil || item.ChannelID <= 0 {
			continue
		}
		snapshot.ChannelStats = append(snapshot.ChannelStats, &models.ChannelStats{
			ChannelID:   item.ChannelID,
			ChannelName: strings.TrimSpace(item.ChannelName),
		})

		modelStats, err := t.channelModelStatsRepo.ListByChannelID(ctx, item.ChannelID)
		if err != nil {
			return nil, err
		}
		for _, modelItem := range modelStats {
			if modelItem == nil || modelItem.ChannelID <= 0 {
				continue
			}
			model := strings.TrimSpace(modelItem.Model)
			if model == "" {
				continue
			}
			key := channelModelKey{
				channelID: modelItem.ChannelID,
				model:     model,
			}
			if _, exists := seenModelKeys[key]; exists {
				continue
			}
			seenModelKeys[key] = struct{}{}
			snapshot.ChannelModelStats = append(snapshot.ChannelModelStats, &models.ChannelModelStats{
				ChannelID: modelItem.ChannelID,
				Model:     model,
			})
		}
	}

	return snapshot, nil
}

// refreshObservationWindowsOnly 仅刷新已有统计记录的观测窗口，不变更累计统计值。
func (t *StatsAggregationTask) refreshObservationWindowsOnly(
	ctx context.Context,
	snapshot *aggregationSnapshot,
	now time.Time,
) error {
	if snapshot == nil {
		return nil
	}
	if t.tm == nil {
		return errors.New("transaction manager is nil")
	}
	if t.channelStatsRepo == nil {
		return errors.New("channel stats repo is nil")
	}
	if t.channelModelStatsRepo == nil {
		return errors.New("channel model stats repo is nil")
	}

	return t.tm.Transaction(ctx, func(txCtx context.Context) error {
		if err := t.applyObservationWindows(txCtx, snapshot, now); err != nil {
			return err
		}

		for _, delta := range snapshot.ChannelStats {
			if delta == nil {
				continue
			}
			merged, err := t.mergeChannelStats(txCtx, delta, nil)
			if err != nil {
				return err
			}
			if err := t.channelStatsRepo.Upsert(txCtx, merged); err != nil {
				return err
			}
		}

		for _, delta := range snapshot.ChannelModelStats {
			if delta == nil {
				continue
			}
			merged, err := t.mergeChannelModelStats(txCtx, delta, nil)
			if err != nil {
				return err
			}
			if err := t.channelModelStatsRepo.Upsert(txCtx, merged); err != nil {
				return err
			}
		}

		return nil
	})
}

// applyObservationWindows 为快照中的渠道和渠道-模型统计填充观测窗口数据。
func (t *StatsAggregationTask) applyObservationWindows(
	ctx context.Context,
	snapshot *aggregationSnapshot,
	now time.Time,
) error {
	if snapshot == nil {
		return nil
	}

	// 计算观测窗口范围
	windowStart, windowEnd := observationWindowRange(now)

	// 提取快照中所有有效的渠道 ID
	channelIDs := make([]int64, 0, len(snapshot.ChannelStats))
	for _, item := range snapshot.ChannelStats {
		if item == nil || item.ChannelID <= 0 {
			continue
		}
		channelIDs = append(channelIDs, item.ChannelID)
	}
	channelIDs = uniqueInt64(channelIDs) // 去重

	// 分别为渠道和渠道-模型维度准备存放日志的映射
	channelLogsMap := make(map[int64][]*models.RequestLog, len(channelIDs))
	channelModelLogsMap := make(map[channelModelKey][]*models.RequestLog, len(snapshot.ChannelModelStats))

	// 如果存在渠道 ID，则从仓库中查询这些渠道在窗口内的所有日志
	windowLogs, err := t.channelStatsRepo.ListRequestLogsByRange(ctx, windowStart, windowEnd)
	if err != nil {
		return err
	}
	windowLogs = logsNormalize(windowLogs)
	t.logger.Info("loaded windowLogs", zap.Int("numLogs", len(windowLogs)), zap.Time("windowStart", windowStart), zap.Time("windowEnd", windowEnd))
	// 将查询出的日志分配到对应的映射中
	for _, item := range windowLogs {
		if item == nil {
			continue
		}
		channelLogsMap[item.ChannelID] = append(channelLogsMap[item.ChannelID], item)
		model := strings.TrimSpace(item.UpstreamModel)
		if model == "" {
			continue
		}
		key := channelModelKey{
			channelID: item.ChannelID,
			model:     model,
		}
		channelModelLogsMap[key] = append(channelModelLogsMap[key], item)
	}
	//if len(channelIDs) > 0 {
	//	windowLogs, err := t.channelStatsRepo.ListRequestLogsByChannelIDsAndRange(ctx, channelIDs, windowStart, windowEnd)
	//	if err != nil {
	//		return err
	//	}
	//	windowLogs = logsNormalize(windowLogs)
	//	t.logger.Info("loaded windowLogs", zap.Int("numLogs", len(windowLogs)), zap.Time("windowStart", windowStart), zap.Time("windowEnd", windowEnd))
	//	// 将查询出的日志分配到对应的映射中
	//	for _, item := range windowLogs {
	//		if item == nil {
	//			continue
	//		}
	//		channelLogsMap[item.ChannelID] = append(channelLogsMap[item.ChannelID], item)
	//		model := strings.TrimSpace(item.UpstreamModel)
	//		if model == "" {
	//			continue
	//		}
	//		key := channelModelKey{
	//			channelID: item.ChannelID,
	//			model:     model,
	//		}
	//		channelModelLogsMap[key] = append(channelModelLogsMap[key], item)
	//	}
	//}

	// 为快照中的每个渠道统计构建观测窗口
	for _, item := range snapshot.ChannelStats {
		if item == nil {
			continue
		}
		item.Window3H = buildObservationWindow3H(now, channelLogsMap[item.ChannelID])
	}
	// 为快照中的每个渠道-模型统计构建观测窗口
	for _, item := range snapshot.ChannelModelStats {
		if item == nil {
			continue
		}
		key := channelModelKey{
			channelID: item.ChannelID,
			model:     strings.TrimSpace(item.Model),
		}
		item.Window3H = buildObservationWindow3H(now, channelModelLogsMap[key])
	}
	return nil
}

// persistMergedStats 将快照中的统计数据与数据库已有数据进行合并并持久化。
func (t *StatsAggregationTask) persistMergedStats(ctx context.Context, snapshot *aggregationSnapshot) error {
	if snapshot == nil {
		return nil
	}
	// 检查必要的仓库依赖
	if t.channelStatsRepo == nil {
		return errors.New("channel stats repo is nil")
	}
	if t.channelModelStatsRepo == nil {
		return errors.New("channel model stats repo is nil")
	}
	if t.tokenStatsRepo == nil {
		return errors.New("token stats repo is nil")
	}
	if t.userStatsRepo == nil {
		return errors.New("user stats repo is nil")
	}
	if t.userUsageDailyStatsRepo == nil {
		return errors.New("user usage daily stats repo is nil")
	}
	if t.userUsageHourlyStatsRepo == nil {
		return errors.New("user usage hourly stats repo is nil")
	}

	// 处理渠道统计
	for _, delta := range snapshot.ChannelStats {
		if delta == nil {
			continue
		}
		// 获取本批次的增量计数器（用于计算 AvgTTFT 和 AvgTPS）
		counter := snapshot.channelCounterMap[delta.ChannelID]
		merged, err := t.mergeChannelStats(ctx, delta, counter)
		if err != nil {
			return err
		}
		// Upsert 合并后的渠道统计
		if err := t.channelStatsRepo.Upsert(ctx, merged); err != nil {
			return err
		}
	}

	// 处理渠道-模型统计
	for _, delta := range snapshot.ChannelModelStats {
		if delta == nil {
			continue
		}
		key := channelModelKey{
			channelID: delta.ChannelID,
			model:     strings.TrimSpace(delta.Model),
		}
		counter := snapshot.channelModelCounterMap[key]
		merged, err := t.mergeChannelModelStats(ctx, delta, counter)
		if err != nil {
			return err
		}
		if err := t.channelModelStatsRepo.Upsert(ctx, merged); err != nil {
			return err
		}
	}

	// 处理 Token 统计
	for _, delta := range snapshot.TokenStats {
		if delta == nil {
			continue
		}
		merged, err := t.mergeTokenStats(ctx, delta)
		if err != nil {
			return err
		}
		if err := t.tokenStatsRepo.Upsert(ctx, merged); err != nil {
			return err
		}
	}

	// 处理用户统计
	for _, delta := range snapshot.UserStats {
		if delta == nil {
			continue
		}
		merged, err := t.mergeUserStats(ctx, delta)
		if err != nil {
			return err
		}
		if err := t.userStatsRepo.Upsert(ctx, merged); err != nil {
			return err
		}
	}

	// 处理用户每日统计
	for _, delta := range snapshot.UserUsageDailyStats {
		if delta == nil {
			continue
		}
		merged, err := t.mergeUserUsageDailyStats(ctx, delta)
		if err != nil {
			return err
		}
		if err := t.userUsageDailyStatsRepo.Upsert(ctx, merged); err != nil {
			return err
		}
	}

	// 处理用户每小时统计
	for _, delta := range snapshot.UserUsageHourlyStats {
		if delta == nil {
			continue
		}
		merged, err := t.mergeUserUsageHourlyStats(ctx, delta)
		if err != nil {
			return err
		}
		if err := t.userUsageHourlyStatsRepo.Upsert(ctx, merged); err != nil {
			return err
		}
	}
	return nil
}

func (t *StatsAggregationTask) mergeChannelStats(ctx context.Context, delta *models.ChannelStats, deltaCounter *statsCounter) (*models.ChannelStats, error) {
	existing, err := t.channelStatsRepo.GetByChannelID(ctx, delta.ChannelID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		existing = &models.ChannelStats{
			ChannelID: delta.ChannelID,
		}
	}
	if channelName := strings.TrimSpace(delta.ChannelName); channelName != "" {
		existing.ChannelName = channelName
	}

	oldOutput := existing.OutputToken
	oldSuccess := existing.RequestSuccess
	oldAvgTTFT := existing.AvgTTFT
	oldAvgTPS := existing.AvgTPS

	existing.InputToken += delta.InputToken
	existing.OutputToken += delta.OutputToken
	existing.CachedCreationInputTokens += delta.CachedCreationInputTokens
	existing.CachedReadInputTokens += delta.CachedReadInputTokens
	existing.RequestSuccess += delta.RequestSuccess
	existing.RequestFailed += delta.RequestFailed
	existing.TotalCostMicros += delta.TotalCostMicros
	existing.Window3H = delta.Window3H

	if deltaCounter == nil {
		deltaCounter = &statsCounter{
			requestSuccess: delta.RequestSuccess,
		}
	}
	existing.AvgTTFT = mergeAvgInt(oldAvgTTFT, oldSuccess, delta.AvgTTFT, deltaCounter.requestSuccess)
	deltaTransfer := float64(deltaCounter.transferTimeSum + deltaCounter.requestSuccess*1000)
	existing.AvgTPS = mergeAvgTPS(oldOutput, oldAvgTPS, delta.OutputToken, deltaTransfer)

	return existing, nil
}

func (t *StatsAggregationTask) mergeChannelModelStats(ctx context.Context, delta *models.ChannelModelStats, deltaCounter *statsCounter) (*models.ChannelModelStats, error) {
	existing, err := t.channelModelStatsRepo.GetByChannelModel(ctx, delta.ChannelID, delta.Model)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		existing = &models.ChannelModelStats{
			ChannelID: delta.ChannelID,
			Model:     delta.Model,
		}
	}

	oldOutput := existing.OutputToken
	oldSuccess := existing.RequestSuccess
	oldAvgTTFT := existing.AvgTTFT
	oldAvgTPS := existing.AvgTPS

	existing.InputToken += delta.InputToken
	existing.OutputToken += delta.OutputToken
	existing.CachedCreationInputTokens += delta.CachedCreationInputTokens
	existing.CachedReadInputTokens += delta.CachedReadInputTokens
	existing.RequestSuccess += delta.RequestSuccess
	existing.RequestFailed += delta.RequestFailed
	existing.TotalCostMicros += delta.TotalCostMicros
	existing.Window3H = delta.Window3H

	if deltaCounter == nil {
		deltaCounter = &statsCounter{
			requestSuccess: delta.RequestSuccess,
		}
	}
	existing.AvgTTFT = mergeAvgInt(oldAvgTTFT, oldSuccess, delta.AvgTTFT, deltaCounter.requestSuccess)
	deltaTransfer := float64(deltaCounter.transferTimeSum)
	existing.AvgTPS = mergeAvgTPS(oldOutput, oldAvgTPS, delta.OutputToken, deltaTransfer)

	return existing, nil
}

func (t *StatsAggregationTask) mergeTokenStats(ctx context.Context, delta *models.TokenStats) (*models.TokenStats, error) {
	existing, err := t.tokenStatsRepo.GetByTokenID(ctx, delta.TokenID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		existing = &models.TokenStats{
			TokenID: delta.TokenID,
		}
	}

	existing.InputToken += delta.InputToken
	existing.OutputToken += delta.OutputToken
	existing.CachedCreationInputTokens += delta.CachedCreationInputTokens
	existing.CachedReadInputTokens += delta.CachedReadInputTokens
	existing.RequestSuccess += delta.RequestSuccess
	existing.RequestFailed += delta.RequestFailed
	existing.TotalCostMicros += delta.TotalCostMicros
	return existing, nil
}

func (t *StatsAggregationTask) mergeUserStats(ctx context.Context, delta *models.UserStats) (*models.UserStats, error) {
	existing, err := t.userStatsRepo.GetByUserID(ctx, delta.UserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		existing = &models.UserStats{
			UserID: delta.UserID,
		}
	}

	existing.InputToken += delta.InputToken
	existing.OutputToken += delta.OutputToken
	existing.CachedCreationInputTokens += delta.CachedCreationInputTokens
	existing.CachedReadInputTokens += delta.CachedReadInputTokens
	existing.RequestSuccess += delta.RequestSuccess
	existing.RequestFailed += delta.RequestFailed
	existing.TotalCostMicros += delta.TotalCostMicros
	return existing, nil
}

func (t *StatsAggregationTask) mergeUserUsageDailyStats(ctx context.Context, delta *models.UserUsageDailyStats) (*models.UserUsageDailyStats, error) {
	existing, err := t.userUsageDailyStatsRepo.GetByUserDate(ctx, delta.UserID, delta.Date)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		existing = &models.UserUsageDailyStats{
			ID:     delta.ID,
			UserID: delta.UserID,
			Date:   delta.Date,
		}
		if existing.ID <= 0 {
			existing.ID = t.nextID()
		}
	}

	existing.InputToken += delta.InputToken
	existing.OutputToken += delta.OutputToken
	existing.CachedCreationInputTokens += delta.CachedCreationInputTokens
	existing.CachedReadInputTokens += delta.CachedReadInputTokens
	existing.RequestSuccess += delta.RequestSuccess
	existing.RequestFailed += delta.RequestFailed
	existing.TotalCostMicros += delta.TotalCostMicros
	return existing, nil
}

func (t *StatsAggregationTask) mergeUserUsageHourlyStats(ctx context.Context, delta *models.UserUsageHourlyStats) (*models.UserUsageHourlyStats, error) {
	existing, err := t.userUsageHourlyStatsRepo.GetByUserDateHour(ctx, delta.UserID, delta.Date, delta.Hour)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		existing = &models.UserUsageHourlyStats{
			ID:     delta.ID,
			UserID: delta.UserID,
			Date:   delta.Date,
			Hour:   delta.Hour,
		}
		if existing.ID <= 0 {
			existing.ID = t.nextID()
		}
	}

	existing.InputToken += delta.InputToken
	existing.OutputToken += delta.OutputToken
	existing.CachedCreationInputTokens += delta.CachedCreationInputTokens
	existing.CachedReadInputTokens += delta.CachedReadInputTokens
	existing.RequestSuccess += delta.RequestSuccess
	existing.RequestFailed += delta.RequestFailed
	existing.TotalCostMicros += delta.TotalCostMicros
	return existing, nil
}

func mergeAvgInt(oldAvg int, oldCount int64, deltaAvg int, deltaCount int64) int {
	if oldCount <= 0 && deltaCount <= 0 {
		return 0
	}
	if oldCount <= 0 {
		return deltaAvg
	}
	if deltaCount <= 0 {
		return oldAvg
	}
	numerator := float64(oldAvg)*float64(oldCount) + float64(deltaAvg)*float64(deltaCount)
	denominator := float64(oldCount + deltaCount)
	if denominator <= 0 {
		return 0
	}
	return int(numerator / denominator)
}

func mergeAvgTPS(oldOutput int64, oldAvgTPS float64, deltaOutput int64, deltaTransfer float64) float64 {
	totalOutput := oldOutput + deltaOutput
	if totalOutput <= 0 {
		return 0
	}
	oldTransfer := estimateTransferFromAvgTPS(oldOutput, oldAvgTPS)
	denominator := oldTransfer + deltaTransfer
	if denominator <= 0 {
		return 0
	}
	return float64(totalOutput) * 1000 / denominator
}

func estimateTransferFromAvgTPS(outputTokens int64, avgTPS float64) float64 {
	if outputTokens <= 0 || avgTPS <= 0 {
		return 0
	}
	return float64(outputTokens) * 1000 / avgTPS
}

func (t *StatsAggregationTask) updateRuntimeState(
	startedAt time.Time,
	now time.Time,
	snapshot *aggregationSnapshot,
	processedLogs int,
	startID int64,
	endID int64,
	taskID int64,
	runErr error,
) {
	finishedAt := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	t.stats.LastRunAt = cloneTimePtr(&finishedAt)
	t.stats.LastDuration = finishedAt.Sub(startedAt)
	t.stats.ProcessedLogs = processedLogs
	t.stats.UpdatedAt = cloneTimePtr(&finishedAt)
	t.stats.LastObservationWindow = currentObservationWindowText(now)
	t.stats.LastStartID = startID
	t.stats.LastEndID = endID
	t.stats.LastTaskID = taskID

	if runErr != nil {
		t.stats.LastError = runErr.Error()
		if t.logger != nil {
			t.logger.Error("stats aggregation task failed",
				zap.Error(runErr),
				zap.Int64("start_id", startID),
				zap.Int64("end_id", endID),
				zap.Int64("task_id", taskID),
			)
		}
		return
	}

	t.stats.LastError = ""
	if snapshot != nil {
		t.stats.ChannelStats = len(snapshot.ChannelStats)
		t.stats.ChannelModelStats = len(snapshot.ChannelModelStats)
		t.stats.TokenStats = len(snapshot.TokenStats)
		t.stats.UserStats = len(snapshot.UserStats)
		t.stats.UserUsageDailyStats = len(snapshot.UserUsageDailyStats)
		t.stats.UserUsageHourlyStats = len(snapshot.UserUsageHourlyStats)
	}
	if t.logger != nil {
		t.logger.Info("stats aggregation task finished",
			zap.Int("logs", processedLogs),
			zap.Int("channel_stats", t.stats.ChannelStats),
			zap.Int("channel_model_stats", t.stats.ChannelModelStats),
			zap.Int("token_stats", t.stats.TokenStats),
			zap.Int("user_stats", t.stats.UserStats),
			zap.Int("daily_stats", t.stats.UserUsageDailyStats),
			zap.Int("hourly_stats", t.stats.UserUsageHourlyStats),
			zap.Int64("start_id", startID),
			zap.Int64("end_id", endID),
			zap.Int64("task_id", taskID),
			zap.Duration("duration", t.stats.LastDuration),
		)
	}
}

func currentObservationWindowText(now time.Time) string {
	start, end := observationWindowRange(now)
	return start.Format(time.RFC3339) + "~" + end.Format(time.RFC3339)
}

func buildObservationWindow3H(now time.Time, logs []*models.RequestLog) models.ObservationWindow3H {
	windowStart, windowEnd := observationWindowRange(now)
	window := models.NewObservationWindow3H()
	window.Buckets = make([]models.ObservationBucket15M, models.ObservationBucketCount)

	bucketDuration := time.Duration(models.ObservationBucket15MMinutes) * time.Minute
	for i := 0; i < models.ObservationBucketCount; i++ {
		startAt := windowStart.Add(time.Duration(i) * bucketDuration)
		window.Buckets[i] = models.ObservationBucket15M{
			StartAt: startAt,
			EndAt:   startAt.Add(bucketDuration),
		}
	}

	bucketCounters := make([]statsCounter, models.ObservationBucketCount)
	for _, item := range logs {
		if item == nil {
			continue
		}
		createdAt := item.CreatedAt
		if createdAt.Before(windowStart) || !createdAt.Before(windowEnd) {
			continue
		}
		idx := int(createdAt.Sub(windowStart) / bucketDuration)
		if idx < 0 || idx >= models.ObservationBucketCount {
			continue
		}
		bucketCounters[idx].add(item)
	}

	for i := 0; i < models.ObservationBucketCount; i++ {
		counter := bucketCounters[i]
		bucket := &window.Buckets[i]
		bucket.InputToken = counter.inputToken
		bucket.OutputToken = counter.outputToken
		bucket.CachedCreationInputTokens = counter.cachedCreationInputTokens
		bucket.CachedReadInputTokens = counter.cachedReadInputTokens
		bucket.RequestSuccess = counter.requestSuccess
		bucket.RequestFailed = counter.requestFailed
		bucket.TotalCostMicros = counter.totalCostMicros
		bucket.AvgTTFT = counter.avgTTFT()
		bucket.AvgTPS = counter.avgTPS()
	}

	return window
}

func observationWindowRange(now time.Time) (time.Time, time.Time) {
	bucketDuration := time.Duration(models.ObservationBucket15MMinutes) * time.Minute
	alignedBucketStart := now.Truncate(bucketDuration)
	windowStart := alignedBucketStart.Add(-time.Duration(models.ObservationBucketCount-1) * bucketDuration)
	windowEnd := alignedBucketStart.Add(bucketDuration)
	return windowStart, windowEnd
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func uniqueInt64(values []int64) []int64 {
	if len(values) <= 1 {
		return values
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// logsNormalize 将日志中的重试记录展开为独立的失败日志条目，
// 实现日志扁平化，便于按渠道、模型等维度准确统计失败次数。
func logsNormalize(logs []*models.RequestLog) []*models.RequestLog {
	// 创建新切片，预分配容量
	data := make([]*models.RequestLog, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			continue // 跳过空日志
		}
		// 保留原始日志（可能最终成功或失败）
		data = append(data, log)

		// 如果没有重试记录，跳过展开
		if len(log.Extra.RetryTrace) <= 0 {
			continue
		}

		// 遍历每一条重试记录
		for _, retry := range log.Extra.RetryTrace {
			// 拷贝原始日志的值，得到一个新的日志结构体
			temp := *log
			cloneLog := &temp

			// 将克隆日志的渠道和模型更新为重试时使用的上游信息
			cloneLog.UpstreamModel = retry.UpstreamModel
			cloneLog.ChannelID = retry.ChannelID

			// 清零所有消耗相关字段，因为这次重试尝试本身不产生计费或 token 消耗
			cloneLog.InputToken = 0
			cloneLog.OutputToken = 0
			cloneLog.CachedCreationInputTokens = 0
			cloneLog.CachedReadInputTokens = 0
			cloneLog.CostMicros = 0
			cloneLog.TransferTime = 0
			cloneLog.TTFT = 0

			// 强制将状态设为失败（重试本质上是一次失败尝试）
			cloneLog.Status = models.RequestStatusFail
			// 记录错误码和错误详情，便于分析失败原因
			cloneLog.ErrorCode = string(retry.StatusCode)
			cloneLog.ErrorMsg = retry.StatusBody

			// 将这次重试对应的失败日志追加到结果集
			data = append(data, cloneLog)
		}
	}
	// 返回扁平化后的日志集合
	return data
}
