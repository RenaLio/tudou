package tasks

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/repository"
	"go.uber.org/zap"
)

func TestStatsCleanupTask_Run_ByConfirmedBoundaries(t *testing.T) {
	channelRepo := &testStatsCleanupChannelRepo{
		items: []*models.Channel{
			{ID: 1, Model: "gpt-4o", CustomModel: "m1"},
			{ID: 2, Model: "c1"},
		},
	}
	channelStatsRepo := &testStatsCleanupChannelStatsRepo{
		items: []*models.ChannelStats{
			{ChannelID: 1},
			{ChannelID: 3},
		},
	}
	channelModelStatsRepo := &testStatsCleanupChannelModelStatsRepo{
		items: []*models.ChannelModelStats{
			{ChannelID: 1, Model: "gpt-4o"},
			{ChannelID: 1, Model: "old-x"}, // 仍保留（同渠道，不按模型细删）
			{ChannelID: 3, Model: "x"},     // orphan
		},
	}
	aiModelRepo := &testStatsCleanupAIModelRepo{
		items: []*models.AIModel{
			{ID: 11, Name: "gpt-4o"},
			{ID: 12, Name: "m1"},
			{ID: 13, Name: "unused-model"},
		},
	}
	requestLogRepo := &testStatsCleanupRequestLogRepo{
		deletedRows: 123,
	}

	task := NewStatsCleanupTask(
		&log.Logger{Logger: zap.NewNop()},
		channelRepo,
		channelStatsRepo,
		channelModelStatsRepo,
		aiModelRepo,
		requestLogRepo,
	)
	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !slices.Equal(channelStatsRepo.deletedChannelIDs, []int64{3}) {
		t.Fatalf("unexpected deleted channel stats ids: %+v", channelStatsRepo.deletedChannelIDs)
	}
	if !slices.Equal(channelModelStatsRepo.deletedByChannelIDs, []int64{3}) {
		t.Fatalf("unexpected deleted channel model stats ids: %+v", channelModelStatsRepo.deletedByChannelIDs)
	}
	if !slices.Equal(aiModelRepo.deletedIDs, []int64{13}) {
		t.Fatalf("unexpected deleted ai model ids: %+v", aiModelRepo.deletedIDs)
	}
	if requestLogRepo.before.IsZero() {
		t.Fatalf("expected request log cutoff to be passed")
	}

	now := time.Now()
	minCutoff := now.AddDate(0, 0, -46)
	maxCutoff := now.AddDate(0, 0, -44)
	if requestLogRepo.before.Before(minCutoff) || requestLogRepo.before.After(maxCutoff) {
		t.Fatalf("unexpected request log cutoff: %s", requestLogRepo.before.Format(time.RFC3339))
	}

	statsAny, err := task.CurrentStats()
	if err != nil {
		t.Fatalf("CurrentStats failed: %v", err)
	}
	stats := statsAny.(StatsCleanupTaskStats)
	if stats.TotalChannels != 2 || stats.TotalChannelStats != 2 || stats.TotalChannelModelStats != 3 || stats.TotalAIModels != 3 {
		t.Fatalf("unexpected totals: %+v", stats)
	}
	if stats.DeletedChannelStats != 1 || stats.DeletedChannelModelStats != 1 || stats.DeletedAIModels != 1 || stats.DeletedRequestLogs != 123 {
		t.Fatalf("unexpected deleted counters: %+v", stats)
	}
}

func TestStatsCleanupTask_Run_NoData(t *testing.T) {
	task := NewStatsCleanupTask(
		&log.Logger{Logger: zap.NewNop()},
		&testStatsCleanupChannelRepo{},
		&testStatsCleanupChannelStatsRepo{},
		&testStatsCleanupChannelModelStatsRepo{},
		&testStatsCleanupAIModelRepo{},
		&testStatsCleanupRequestLogRepo{},
	)
	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	statsAny, err := task.CurrentStats()
	if err != nil {
		t.Fatalf("CurrentStats failed: %v", err)
	}
	stats := statsAny.(StatsCleanupTaskStats)
	if stats.TotalChannels != 0 || stats.TotalAIModels != 0 || stats.DeletedRequestLogs != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

type testStatsCleanupChannelRepo struct {
	items []*models.Channel
}

func (r *testStatsCleanupChannelRepo) List(_ context.Context, _ repository.ChannelListOption) ([]*models.Channel, int64, error) {
	out := make([]*models.Channel, 0, len(r.items))
	for _, item := range r.items {
		cp := *item
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

type testStatsCleanupChannelStatsRepo struct {
	items             []*models.ChannelStats
	deletedChannelIDs []int64
}

func (r *testStatsCleanupChannelStatsRepo) ListAll(_ context.Context) ([]*models.ChannelStats, error) {
	out := make([]*models.ChannelStats, 0, len(r.items))
	for _, item := range r.items {
		cp := *item
		out = append(out, &cp)
	}
	return out, nil
}

func (r *testStatsCleanupChannelStatsRepo) DeleteByChannelIDs(_ context.Context, channelIDs []int64) (int64, error) {
	ids := append([]int64(nil), channelIDs...)
	slices.Sort(ids)
	r.deletedChannelIDs = ids

	set := make(map[int64]struct{}, len(channelIDs))
	for _, id := range channelIDs {
		set[id] = struct{}{}
	}
	var affected int64
	for _, item := range r.items {
		if item == nil {
			continue
		}
		if _, ok := set[item.ChannelID]; ok {
			affected++
		}
	}
	return affected, nil
}

type testStatsCleanupChannelModelStatsRepo struct {
	items               []*models.ChannelModelStats
	deletedByChannelIDs []int64
}

func (r *testStatsCleanupChannelModelStatsRepo) ListAll(_ context.Context) ([]*models.ChannelModelStats, error) {
	out := make([]*models.ChannelModelStats, 0, len(r.items))
	for _, item := range r.items {
		cp := *item
		out = append(out, &cp)
	}
	return out, nil
}

func (r *testStatsCleanupChannelModelStatsRepo) DeleteByChannelIDs(_ context.Context, channelIDs []int64) (int64, error) {
	ids := append([]int64(nil), channelIDs...)
	slices.Sort(ids)
	r.deletedByChannelIDs = ids

	set := make(map[int64]struct{}, len(channelIDs))
	for _, id := range channelIDs {
		set[id] = struct{}{}
	}
	var affected int64
	for _, item := range r.items {
		if item == nil {
			continue
		}
		if _, ok := set[item.ChannelID]; ok {
			affected++
		}
	}
	return affected, nil
}

type testStatsCleanupAIModelRepo struct {
	items      []*models.AIModel
	deletedIDs []int64
}

func (r *testStatsCleanupAIModelRepo) List(_ context.Context, _ repository.AIModelListOption) ([]*models.AIModel, int64, error) {
	out := make([]*models.AIModel, 0, len(r.items))
	for _, item := range r.items {
		cp := *item
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

func (r *testStatsCleanupAIModelRepo) DeleteByIDs(_ context.Context, ids []int64) (int64, error) {
	cp := append([]int64(nil), ids...)
	slices.Sort(cp)
	r.deletedIDs = cp
	return int64(len(cp)), nil
}

type testStatsCleanupRequestLogRepo struct {
	before      time.Time
	deletedRows int64
}

func (r *testStatsCleanupRequestLogRepo) DeleteBefore(_ context.Context, before time.Time) (int64, error) {
	r.before = before
	return r.deletedRows, nil
}
