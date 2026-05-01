package tasks

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/store"
	"go.uber.org/zap"
)

type PriceSyncTask struct {
	logger     *log.Logger
	priceStore *store.ModelPriceStore
	repo       repository.AIModelRepo
	mu         sync.RWMutex
	stats      PriceSyncTaskStats
}

type PriceSyncTaskStats struct {
	LastRunAt     *time.Time    `json:"lastRunAt,omitempty"`
	LastDuration  time.Duration `json:"lastDuration"`
	LastError     string        `json:"lastError,omitempty"`
	TotalModels   int           `json:"totalModels"`
	SyncedModels  int           `json:"syncedModels"`
	UpdatedModels int           `json:"updatedModels"`
	SkippedModels int           `json:"skippedModels"`
	UpdatedAt     *time.Time    `json:"updatedAt,omitempty"`
}

func NewPriceSyncTask(
	logger *log.Logger,
	priceStore *store.ModelPriceStore,
	repo repository.AIModelRepo,
) *PriceSyncTask {
	return &PriceSyncTask{
		logger:     logger,
		priceStore: priceStore,
		repo:       repo,
	}
}

func (t *PriceSyncTask) Name() string {
	return PriceSyncTaskName
}

func (t *PriceSyncTask) CurrentStats() (any, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cloned := t.stats
	cloned.LastRunAt = cloneTimePtr(t.stats.LastRunAt)
	cloned.UpdatedAt = cloneTimePtr(t.stats.UpdatedAt)
	return cloned, nil
}

func (t *PriceSyncTask) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	startedAt := time.Now()
	if t.repo == nil {
		err := errors.New("ai model repo is nil")
		t.updateRuntimeState(startedAt, 0, 0, 0, 0, err)
		return err
	}
	if t.priceStore == nil {
		err := errors.New("model price store is nil")
		t.updateRuntimeState(startedAt, 0, 0, 0, 0, err)
		return err
	}

	t.priceStore.TryRefresh()

	total := 0
	synced := 0
	updated := 0
	skipped := 0

	page := 1
	pageSize := repository.GetMaxPageSize()
	if pageSize <= 0 {
		pageSize = 200
	}

	for {
		modelsList, _, err := t.repo.List(ctx, repository.AIModelListOption{
			Page:     page,
			PageSize: pageSize,
			OrderBy:  "id ASC",
		})
		if err != nil {
			t.updateRuntimeState(startedAt, total, synced, updated, skipped, err)
			return err
		}
		if len(modelsList) == 0 {
			break
		}

		for _, model := range modelsList {
			total++
			if model == nil {
				skipped++
				continue
			}
			if model.Extra.DisableSync {
				skipped++
				continue
			}

			syncPath := strings.TrimSpace(model.Extra.SyncModelInfoPath)
			if syncPath == "" {
				syncPath = strings.TrimSpace(t.priceStore.FindSimilarPath("", model.Name))
			}
			if syncPath == "" || !t.priceStore.HasPath(syncPath) {
				skipped++
				continue
			}

			nextPricing := models.ModelPricing{
				InputPrice:               t.priceStore.GetInputPrice(syncPath),
				OutputPrice:              t.priceStore.GetOutputPrice(syncPath),
				CacheReadPrice:           t.priceStore.GetCacheReadPrice(syncPath),
				CacheCreatePrice:         t.priceStore.GetCacheCreatePrice(syncPath),
				Over200KInputPrice:       t.priceStore.GetOver200KInputPrice(syncPath),
				Over200KOutputPrice:      t.priceStore.GetOver200KOutputPrice(syncPath),
				Over200KCacheReadPrice:   t.priceStore.GetOver200KCacheReadPrice(syncPath),
				Over200KCacheCreatePrice: t.priceStore.GetOver200KCacheWritePrice(syncPath),
			}

			synced++
			if pricingEqual(model.Pricing, nextPricing) && strings.TrimSpace(model.Extra.SyncModelInfoPath) == syncPath {
				continue
			}

			model.Pricing = nextPricing
			model.Extra.SyncModelInfoPath = syncPath
			if err = t.repo.Update(ctx, model); err != nil {
				t.updateRuntimeState(startedAt, total, synced, updated, skipped, err)
				return err
			}
			updated++
		}

		page++
	}

	t.updateRuntimeState(startedAt, total, synced, updated, skipped, nil)
	return nil
}

func (t *PriceSyncTask) updateRuntimeState(
	startedAt time.Time,
	total int,
	synced int,
	updated int,
	skipped int,
	runErr error,
) {
	now := time.Now()
	lastRunAt := startedAt
	stats := PriceSyncTaskStats{
		LastRunAt:     &lastRunAt,
		LastDuration:  now.Sub(startedAt),
		TotalModels:   total,
		SyncedModels:  synced,
		UpdatedModels: updated,
		SkippedModels: skipped,
		UpdatedAt:     &now,
	}
	if runErr != nil {
		stats.LastError = runErr.Error()
	} else {
		stats.LastError = ""
	}

	t.mu.Lock()
	t.stats = stats
	t.mu.Unlock()

	if t.logger == nil {
		return
	}
	if runErr != nil {
		t.logger.Error(
			"price sync task failed",
			zap.Int("total_models", total),
			zap.Int("synced_models", synced),
			zap.Int("updated_models", updated),
			zap.Int("skipped_models", skipped),
			zap.Error(runErr),
		)
		return
	}

	t.logger.Info(
		"price sync task finished",
		zap.Int("total_models", total),
		zap.Int("synced_models", synced),
		zap.Int("updated_models", updated),
		zap.Int("skipped_models", skipped),
		zap.Duration("duration", stats.LastDuration),
	)
}

func pricingEqual(a, b models.ModelPricing) bool {
	return a.InputPrice == b.InputPrice &&
		a.OutputPrice == b.OutputPrice &&
		a.CacheCreatePrice == b.CacheCreatePrice &&
		a.CacheReadPrice == b.CacheReadPrice &&
		a.PerRequestPrice == b.PerRequestPrice &&
		a.Over200KInputPrice == b.Over200KInputPrice &&
		a.Over200KOutputPrice == b.Over200KOutputPrice &&
		a.Over200KCacheCreatePrice == b.Over200KCacheCreatePrice &&
		a.Over200KCacheReadPrice == b.Over200KCacheReadPrice &&
		a.Over200KPerRequestPrice == b.Over200KPerRequestPrice
}
