package tasks

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"go.uber.org/zap"
)

type ChannelModelSyncFetcher interface {
	FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error)
}

type ChannelModelSyncChannelService interface {
	List(ctx context.Context, req v1.ListChannelsRequest) (*v1.ListResponse[v1.ChannelResponse], error)
	Update(ctx context.Context, id int64, req v1.UpdateChannelRequest) (*v1.ChannelResponse, error)
}

type ChannelModelSyncTask struct {
	logger     *log.Logger
	channelSVC ChannelModelSyncChannelService
	fetcher    ChannelModelSyncFetcher

	mu    sync.RWMutex
	stats ChannelModelSyncTaskStats
}

type ChannelModelSyncTaskStats struct {
	LastRunAt       *time.Time    `json:"lastRunAt,omitempty"`
	LastDuration    time.Duration `json:"lastDuration"`
	LastError       string        `json:"lastError,omitempty"`
	TotalChannels   int           `json:"totalChannels"`
	SyncEnabled     int           `json:"syncEnabled"`
	UpdatedChannels int           `json:"updatedChannels"`
	SkippedChannels int           `json:"skippedChannels"`
	FailedChannels  int           `json:"failedChannels"`
	UpdatedAt       *time.Time    `json:"updatedAt,omitempty"`
}

func NewChannelModelSyncTask(
	logger *log.Logger,
	channelSVC ChannelModelSyncChannelService,
	fetcher ChannelModelSyncFetcher,
) *ChannelModelSyncTask {
	return &ChannelModelSyncTask{
		logger:     logger,
		channelSVC: channelSVC,
		fetcher:    fetcher,
	}
}

func (t *ChannelModelSyncTask) Name() string {
	return ChannelModelSyncTaskName
}

func (t *ChannelModelSyncTask) CurrentStats() (any, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cloned := t.stats
	cloned.LastRunAt = cloneTimePtr(t.stats.LastRunAt)
	cloned.UpdatedAt = cloneTimePtr(t.stats.UpdatedAt)
	return cloned, nil
}

func (t *ChannelModelSyncTask) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	startedAt := time.Now()
	if t.channelSVC == nil {
		err := errors.New("channel service is nil")
		t.updateRuntimeState(startedAt, 0, 0, 0, 0, 0, err)
		return err
	}
	if t.fetcher == nil {
		err := errors.New("channel model fetcher is nil")
		t.updateRuntimeState(startedAt, 0, 0, 0, 0, 0, err)
		return err
	}

	total := 0
	enabled := 0
	updated := 0
	skipped := 0
	failed := 0

	page := 1
	pageSize := 200

	for {
		listResp, err := t.channelSVC.List(ctx, v1.ListChannelsRequest{
			Page:     page,
			PageSize: pageSize,
			OrderBy:  "id ASC",
		})
		if err != nil {
			t.updateRuntimeState(startedAt, total, enabled, updated, skipped, failed, err)
			return err
		}
		if listResp == nil || len(listResp.Items) == 0 {
			break
		}

		for _, ch := range listResp.Items {
			total++
			if !ch.Settings.AutoSyncUpstreamModels {
				skipped++
				continue
			}
			enabled++

			modelsList, err := t.fetcher.FetchModel(ctx, &v1.FetchModelRequest{
				Type:    ch.Type,
				BaseURL: ch.BaseURL,
				APIKey:  ch.APIKey,
			})
			if err != nil {
				failed++
				if t.logger != nil {
					t.logger.Warn(
						"channel model sync fetch failed",
						zap.Int64("channel_id", ch.ID),
						zap.String("channel_name", ch.Name),
						zap.Error(err),
					)
				}
				continue
			}

			modelsList, err = filterSyncModelList(modelsList, ch.Settings)
			if err != nil {
				failed++
				if t.logger != nil {
					t.logger.Warn(
						"channel model sync filter failed",
						zap.Int64("channel_id", ch.ID),
						zap.String("channel_name", ch.Name),
						zap.Error(err),
					)
				}
				continue
			}

			nextModel := normalizeModelList(modelsList)
			if nextModel == strings.TrimSpace(ch.Model) {
				skipped++
				continue
			}

			if _, err = t.channelSVC.Update(ctx, ch.ID, v1.UpdateChannelRequest{Model: &nextModel}); err != nil {
				failed++
				if t.logger != nil {
					t.logger.Warn(
						"channel model sync update failed",
						zap.Int64("channel_id", ch.ID),
						zap.String("channel_name", ch.Name),
						zap.Error(err),
					)
				}
				continue
			}
			updated++
		}

		page++
	}

	t.updateRuntimeState(startedAt, total, enabled, updated, skipped, failed, nil)
	return nil
}

// filterSyncModelList 仅用于渠道模型定时自动同步。
// 行为顺序：
// 1. 如果配置了白名单，则只保留匹配白名单的模型
// 2. 如果配置了黑名单，则再剔除匹配黑名单的模型
// 3. 任一正则编译失败，视为该渠道本次自动同步失败
// 4. 不在 RelayService.FetchModel 中复用，避免影响手动测试渠道并拉模型列表
func filterSyncModelList(modelsList []string, settings models.ChannelSettings) ([]string, error) {
	whitelistPattern := strings.TrimSpace(settings.SyncModelWhitelistRegex)
	blacklistPattern := strings.TrimSpace(settings.SyncModelBlacklistRegex)

	if whitelistPattern == "" && blacklistPattern == "" {
		return modelsList, nil
	}

	var (
		whitelist *regexp.Regexp
		blacklist *regexp.Regexp
		err       error
	)
	if whitelistPattern != "" {
		whitelist, err = regexp.Compile(whitelistPattern)
		if err != nil {
			return nil, fmt.Errorf("compile sync model whitelist regex: %w", err)
		}
	}
	if blacklistPattern != "" {
		blacklist, err = regexp.Compile(blacklistPattern)
		if err != nil {
			return nil, fmt.Errorf("compile sync model blacklist regex: %w", err)
		}
	}

	filtered := make([]string, 0, len(modelsList))
	for _, name := range modelsList {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if whitelist != nil && !whitelist.MatchString(trimmed) {
			continue
		}
		if blacklist != nil && blacklist.MatchString(trimmed) {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered, nil
}

func normalizeModelList(modelsList []string) string {
	if len(modelsList) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(modelsList))
	items := make([]string, 0, len(modelsList))
	for _, name := range modelsList {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		items = append(items, name)
	}
	if len(items) == 0 {
		return ""
	}
	slices.Sort(items)
	return strings.Join(items, ",")
}

func (t *ChannelModelSyncTask) updateRuntimeState(
	startedAt time.Time,
	total int,
	enabled int,
	updated int,
	skipped int,
	failed int,
	runErr error,
) {
	now := time.Now()
	lastRunAt := startedAt
	stats := ChannelModelSyncTaskStats{
		LastRunAt:       &lastRunAt,
		LastDuration:    now.Sub(startedAt),
		TotalChannels:   total,
		SyncEnabled:     enabled,
		UpdatedChannels: updated,
		SkippedChannels: skipped,
		FailedChannels:  failed,
		UpdatedAt:       &now,
	}
	if runErr != nil {
		stats.LastError = runErr.Error()
	}

	t.mu.Lock()
	t.stats = stats
	t.mu.Unlock()

	if t.logger == nil {
		return
	}

	if runErr != nil {
		t.logger.Error(
			"channel model sync task failed",
			zap.Int("total_channels", total),
			zap.Int("sync_enabled", enabled),
			zap.Int("updated_channels", updated),
			zap.Int("skipped_channels", skipped),
			zap.Int("failed_channels", failed),
			zap.Error(runErr),
		)
		return
	}

	t.logger.Info(
		"channel model sync task finished",
		zap.Int("total_channels", total),
		zap.Int("sync_enabled", enabled),
		zap.Int("updated_channels", updated),
		zap.Int("skipped_channels", skipped),
		zap.Int("failed_channels", failed),
		zap.Duration("duration", stats.LastDuration),
	)
}
