package tasks

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/repository"
	"go.uber.org/zap"
)

const requestLogRetentionDays = 45

type StatsCleanupTask struct {
	logger                *log.Logger
	channelRepo           StatsCleanupChannelRepo
	channelStatsRepo      StatsCleanupChannelStatsRepo
	channelModelStatsRepo StatsCleanupChannelModelStatsRepo
	aiModelRepo           StatsCleanupAIModelRepo
	requestLogRepo        StatsCleanupRequestLogRepo

	mu    sync.RWMutex
	stats StatsCleanupTaskStats
}

type StatsCleanupTaskStats struct {
	LastRunAt                  *time.Time    `json:"lastRunAt,omitempty"`
	LastDuration               time.Duration `json:"lastDuration"`
	LastError                  string        `json:"lastError,omitempty"`
	TotalChannels              int           `json:"totalChannels"`
	TotalChannelStats          int           `json:"totalChannelStats"`
	TotalChannelModelStats     int           `json:"totalChannelModelStats"`
	TotalAIModels              int           `json:"totalAiModels"`
	DeletedChannelStats        int64         `json:"deletedChannelStats"`
	DeletedChannelModelStats   int64         `json:"deletedChannelModelStats"`
	DeletedAIModels            int64         `json:"deletedAiModels"`
	DeletedRequestLogs         int64         `json:"deletedRequestLogs"`
	RequestLogRetentionCutoff  *time.Time    `json:"requestLogRetentionCutoff,omitempty"`
	DeletedInvalidChannelStats int64         `json:"deletedInvalidChannelStats"`
	UpdatedAt                  *time.Time    `json:"updatedAt,omitempty"`
}

type StatsCleanupChannelRepo interface {
	List(ctx context.Context, opt repository.ChannelListOption) ([]*models.Channel, int64, error)
}

type StatsCleanupChannelStatsRepo interface {
	ListAll(ctx context.Context) ([]*models.ChannelStats, error)
	DeleteByChannelIDs(ctx context.Context, channelIDs []int64) (int64, error)
}

type StatsCleanupChannelModelStatsRepo interface {
	ListAll(ctx context.Context) ([]*models.ChannelModelStats, error)
	DeleteByChannelIDs(ctx context.Context, channelIDs []int64) (int64, error)
}

type StatsCleanupAIModelRepo interface {
	List(ctx context.Context, opt repository.AIModelListOption) ([]*models.AIModel, int64, error)
	DeleteByNames(ctx context.Context, names []string) (int64, error)
}

type StatsCleanupRequestLogRepo interface {
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

func NewStatsCleanupTask(
	logger *log.Logger,
	channelRepo StatsCleanupChannelRepo,
	channelStatsRepo StatsCleanupChannelStatsRepo,
	channelModelStatsRepo StatsCleanupChannelModelStatsRepo,
	aiModelRepo StatsCleanupAIModelRepo,
	requestLogRepo StatsCleanupRequestLogRepo,
) *StatsCleanupTask {
	return &StatsCleanupTask{
		logger:                logger,
		channelRepo:           channelRepo,
		channelStatsRepo:      channelStatsRepo,
		channelModelStatsRepo: channelModelStatsRepo,
		aiModelRepo:           aiModelRepo,
		requestLogRepo:        requestLogRepo,
	}
}

func (t *StatsCleanupTask) Name() string {
	return StatsCleanupTaskName
}

func (t *StatsCleanupTask) CurrentStats() (any, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cloned := t.stats
	cloned.LastRunAt = cloneTimePtr(t.stats.LastRunAt)
	cloned.UpdatedAt = cloneTimePtr(t.stats.UpdatedAt)
	cloned.RequestLogRetentionCutoff = cloneTimePtr(t.stats.RequestLogRetentionCutoff)
	return cloned, nil
}

func (t *StatsCleanupTask) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	startedAt := time.Now()
	if t.channelRepo == nil {
		err := errors.New("channel repo is nil")
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}
	if t.channelStatsRepo == nil {
		err := errors.New("channel stats repo is nil")
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}
	if t.channelModelStatsRepo == nil {
		err := errors.New("channel model stats repo is nil")
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}
	if t.aiModelRepo == nil {
		err := errors.New("ai model repo is nil")
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}
	if t.requestLogRepo == nil {
		err := errors.New("request log repo is nil")
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}

	channels, err := t.listAllChannels(ctx)
	if err != nil {
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}
	channelStats, err := t.channelStatsRepo.ListAll(ctx)
	if err != nil {
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}
	channelModelStats, err := t.channelModelStatsRepo.ListAll(ctx)
	if err != nil {
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}
	aiModels, err := t.listAllAIModels(ctx)
	if err != nil {
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}

	channelIDSet := make(map[int64]struct{}, len(channels))
	channelModelNameSet := make(map[string]struct{}, 256)
	for _, ch := range channels {
		if ch == nil || ch.ID <= 0 {
			continue
		}
		channelIDSet[ch.ID] = struct{}{}
		for name := range ch.Models() {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			channelModelNameSet[name] = struct{}{}
		}
	}

	orphanChannelIDs := make(map[int64]struct{})
	for _, item := range channelStats {
		if item == nil || item.ChannelID <= 0 {
			continue
		}
		if _, ok := channelIDSet[item.ChannelID]; !ok {
			orphanChannelIDs[item.ChannelID] = struct{}{}
		}
	}
	for _, item := range channelModelStats {
		if item == nil || item.ChannelID <= 0 {
			continue
		}
		if _, ok := channelIDSet[item.ChannelID]; !ok {
			orphanChannelIDs[item.ChannelID] = struct{}{}
		}
	}

	orphanIDs := mapKeysToSortedSlice(orphanChannelIDs)
	var deletedChannelStats int64
	var deletedChannelModelStats int64
	if len(orphanIDs) > 0 {
		deletedChannelStats, err = t.channelStatsRepo.DeleteByChannelIDs(ctx, orphanIDs)
		if err != nil {
			t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
			return err
		}
		deletedChannelModelStats, err = t.channelModelStatsRepo.DeleteByChannelIDs(ctx, orphanIDs)
		if err != nil {
			t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
			return err
		}
	}

	deleteAIModelNames := make([]string, 0, len(aiModels))
	for _, item := range aiModels {
		if item == nil {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if _, ok := channelModelNameSet[name]; ok {
			continue
		}
		deleteAIModelNames = append(deleteAIModelNames, name)
	}

	var deletedAIModels int64
	if len(deleteAIModelNames) > 0 {
		deletedAIModels, err = t.aiModelRepo.DeleteByNames(ctx, deleteAIModelNames)
		if err != nil {
			t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
			return err
		}
	}

	retentionCutoff := time.Now().AddDate(0, 0, -requestLogRetentionDays)
	deletedRequestLogs, err := t.requestLogRepo.DeleteBefore(ctx, retentionCutoff)
	if err != nil {
		t.updateRuntimeState(startedAt, StatsCleanupTaskStats{}, err)
		return err
	}

	runStats := StatsCleanupTaskStats{
		TotalChannels:              len(channelIDSet),
		TotalChannelStats:          len(channelStats),
		TotalChannelModelStats:     len(channelModelStats),
		TotalAIModels:              len(aiModels),
		DeletedChannelStats:        deletedChannelStats,
		DeletedChannelModelStats:   deletedChannelModelStats,
		DeletedAIModels:            deletedAIModels,
		DeletedRequestLogs:         deletedRequestLogs,
		RequestLogRetentionCutoff:  &retentionCutoff,
		DeletedInvalidChannelStats: deletedChannelStats + deletedChannelModelStats,
	}
	t.updateRuntimeState(startedAt, runStats, nil)
	return nil
}

func (t *StatsCleanupTask) listAllChannels(ctx context.Context) ([]*models.Channel, error) {
	page := 1
	pageSize := 200
	all := make([]*models.Channel, 0, 1024)
	for {
		items, _, err := t.channelRepo.List(ctx, repository.ChannelListOption{
			Page:     page,
			PageSize: pageSize,
			OrderBy:  "id ASC",
		})
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		if len(items) < pageSize {
			break
		}
		page++
	}
	return all, nil
}

func (t *StatsCleanupTask) listAllAIModels(ctx context.Context) ([]*models.AIModel, error) {
	page := 1
	pageSize := 200
	all := make([]*models.AIModel, 0, 1024)
	for {
		items, _, err := t.aiModelRepo.List(ctx, repository.AIModelListOption{
			Page:     page,
			PageSize: pageSize,
			OrderBy:  "id ASC",
		})
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		if len(items) < pageSize {
			break
		}
		page++
	}
	return all, nil
}

func mapKeysToSortedSlice(m map[int64]struct{}) []int64 {
	if len(m) == 0 {
		return nil
	}
	out := make([]int64, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func (t *StatsCleanupTask) updateRuntimeState(
	startedAt time.Time,
	runStats StatsCleanupTaskStats,
	runErr error,
) {
	now := time.Now()
	lastRunAt := startedAt
	runStats.LastRunAt = &lastRunAt
	runStats.LastDuration = now.Sub(startedAt)
	runStats.UpdatedAt = &now
	if runErr != nil {
		runStats.LastError = runErr.Error()
	}

	t.mu.Lock()
	t.stats = runStats
	t.mu.Unlock()

	if t.logger == nil {
		return
	}

	if runErr != nil {
		t.logger.Error(
			"stats cleanup task failed",
			zap.Int("total_channels", runStats.TotalChannels),
			zap.Int("total_channel_stats", runStats.TotalChannelStats),
			zap.Int("total_channel_model_stats", runStats.TotalChannelModelStats),
			zap.Int("total_ai_models", runStats.TotalAIModels),
			zap.Error(runErr),
		)
		return
	}

	t.logger.Info(
		"stats cleanup task finished",
		zap.Int("total_channels", runStats.TotalChannels),
		zap.Int("total_channel_stats", runStats.TotalChannelStats),
		zap.Int("total_channel_model_stats", runStats.TotalChannelModelStats),
		zap.Int("total_ai_models", runStats.TotalAIModels),
		zap.Int64("deleted_channel_stats", runStats.DeletedChannelStats),
		zap.Int64("deleted_channel_model_stats", runStats.DeletedChannelModelStats),
		zap.Int64("deleted_ai_models", runStats.DeletedAIModels),
		zap.Int64("deleted_request_logs", runStats.DeletedRequestLogs),
		zap.Duration("duration", runStats.LastDuration),
	)
}
