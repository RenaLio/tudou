package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestAggregateRequestLogs_BuildsAllStats(t *testing.T) {
	logs := []*models.RequestLog{
		{
			ChannelID:                 101,
			Model:                     "gpt-4o-mini",
			TokenID:                   201,
			UserID:                    301,
			InputToken:                10,
			OutputToken:               20,
			CachedCreationInputTokens: 1,
			CachedReadInputTokens:     0,
			CostMicros:                100,
			Status:                    models.RequestStatusSuccess,
			TTFT:                      100,
			TransferTime:              2000,
			CreatedAt:                 time.Date(2026, 4, 27, 10, 5, 0, 0, time.UTC),
		},
		{
			ChannelID:                 101,
			Model:                     "gpt-4o-mini",
			TokenID:                   201,
			UserID:                    301,
			InputToken:                5,
			OutputToken:               0,
			CachedCreationInputTokens: 0,
			CachedReadInputTokens:     0,
			CostMicros:                50,
			Status:                    models.RequestStatusFail,
			TTFT:                      0,
			TransferTime:              1000,
			CreatedAt:                 time.Date(2026, 4, 27, 10, 10, 0, 0, time.UTC),
		},
		{
			ChannelID:                 101,
			Model:                     "claude-3-5-sonnet",
			TokenID:                   202,
			UserID:                    302,
			InputToken:                7,
			OutputToken:               14,
			CachedCreationInputTokens: 2,
			CachedReadInputTokens:     1,
			CostMicros:                70,
			Status:                    models.RequestStatusSuccess,
			TTFT:                      200,
			TransferTime:              1400,
			CreatedAt:                 time.Date(2026, 4, 27, 11, 1, 0, 0, time.UTC),
		},
	}

	var id int64 = 9000
	snapshot := aggregateRequestLogs(logs, func() int64 {
		id++
		return id
	})

	if len(snapshot.ChannelStats) != 1 {
		t.Fatalf("expected 1 channel stat, got %d", len(snapshot.ChannelStats))
	}
	ch := snapshot.ChannelStats[0]
	if ch.ChannelID != 101 || ch.InputToken != 22 || ch.OutputToken != 34 || ch.RequestSuccess != 2 || ch.RequestFailed != 1 || ch.TotalCostMicros != 220 {
		t.Fatalf("unexpected channel stats: %+v", *ch)
	}
	if ch.AvgTTFT != 150 {
		t.Fatalf("unexpected channel avgTTFT: %d", ch.AvgTTFT)
	}
	if ch.AvgTPS < 5.3124 || ch.AvgTPS > 5.3126 {
		t.Fatalf("unexpected channel avgTPS: %f", ch.AvgTPS)
	}

	if len(snapshot.ChannelModelStats) != 2 {
		t.Fatalf("expected 2 channel-model stats, got %d", len(snapshot.ChannelModelStats))
	}
	modelA := findChannelModelStat(snapshot.ChannelModelStats, 101, "gpt-4o-mini")
	if modelA == nil {
		t.Fatal("missing channel-model stats for gpt-4o-mini")
	}
	if modelA.InputToken != 15 || modelA.OutputToken != 20 || modelA.RequestSuccess != 1 || modelA.RequestFailed != 1 {
		t.Fatalf("unexpected modelA stats: %+v", *modelA)
	}
	if modelA.AvgTTFT != 100 {
		t.Fatalf("unexpected modelA avgTTFT: %d", modelA.AvgTTFT)
	}
	if modelA.AvgTPS < 6.6665 || modelA.AvgTPS > 6.6668 {
		t.Fatalf("unexpected modelA avgTPS: %f", modelA.AvgTPS)
	}

	if len(snapshot.TokenStats) != 2 {
		t.Fatalf("expected 2 token stats, got %d", len(snapshot.TokenStats))
	}
	token1 := findTokenStat(snapshot.TokenStats, 201)
	if token1 == nil || token1.InputToken != 15 || token1.OutputToken != 20 || token1.RequestSuccess != 1 || token1.RequestFailed != 1 {
		t.Fatalf("unexpected token1 stats: %+v", token1)
	}

	if len(snapshot.UserStats) != 2 {
		t.Fatalf("expected 2 user stats, got %d", len(snapshot.UserStats))
	}
	user1 := findUserStat(snapshot.UserStats, 301)
	if user1 == nil || user1.InputToken != 15 || user1.OutputToken != 20 || user1.RequestSuccess != 1 || user1.RequestFailed != 1 {
		t.Fatalf("unexpected user1 stats: %+v", user1)
	}

	if len(snapshot.UserUsageDailyStats) != 2 {
		t.Fatalf("expected 2 daily stats, got %d", len(snapshot.UserUsageDailyStats))
	}
	day1 := findDailyStat(snapshot.UserUsageDailyStats, 301, "2026-04-27")
	if day1 == nil || day1.InputToken != 15 || day1.OutputToken != 20 || day1.RequestSuccess != 1 || day1.RequestFailed != 1 {
		t.Fatalf("unexpected day1 stats: %+v", day1)
	}

	if len(snapshot.UserUsageHourlyStats) != 2 {
		t.Fatalf("expected 2 hourly stats, got %d", len(snapshot.UserUsageHourlyStats))
	}
	hour1 := findHourlyStat(snapshot.UserUsageHourlyStats, 301, "2026-04-27", 10)
	if hour1 == nil || hour1.InputToken != 15 || hour1.OutputToken != 20 || hour1.RequestSuccess != 1 || hour1.RequestFailed != 1 {
		t.Fatalf("unexpected hour1 stats: %+v", hour1)
	}
}

func TestAggregateRequestLogs_UsesUTCForDailyAndHourly(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	logs := []*models.RequestLog{
		{
			ChannelID:    1,
			Model:        "x",
			TokenID:      2,
			UserID:       3,
			InputToken:   1,
			OutputToken:  2,
			CostMicros:   3,
			Status:       models.RequestStatusSuccess,
			TTFT:         10,
			TransferTime: 100,
			CreatedAt:    time.Date(2026, 5, 1, 0, 5, 0, 0, cst), // 2026-04-30 16:05
		},
	}

	snapshot := aggregateRequestLogs(logs, func() int64 { return 10001 })
	day := findDailyStat(snapshot.UserUsageDailyStats, 3, "2026-04-30")
	if day == nil {
		t.Fatal("expected  daily stats key 2026-04-30")
	}
	hour := findHourlyStat(snapshot.UserUsageHourlyStats, 3, "2026-04-30", 16)
	if hour == nil {
		t.Fatal("expected  hourly stats key 16")
	}
}

func TestAggregateRequestLogs_ChannelName(t *testing.T) {
	logs := []*models.RequestLog{
		{
			ChannelID:   101,
			ChannelName: "",
			Status:      models.RequestStatusSuccess,
		},
		{
			ChannelID:   101,
			ChannelName: "Prod-Channel-A",
			Status:      models.RequestStatusSuccess,
		},
	}

	snapshot := aggregateRequestLogs(logs, nil)
	if len(snapshot.ChannelStats) != 1 {
		t.Fatalf("expected 1 channel stat, got %d", len(snapshot.ChannelStats))
	}
	if snapshot.ChannelStats[0].ChannelName != "Prod-Channel-A" {
		t.Fatalf("unexpected channel name: %+v", snapshot.ChannelStats[0].ChannelName)
	}
}

func TestMergeChannelStats_EmptyDeltaChannelNameKeepsExisting(t *testing.T) {
	channelID := int64(101)
	repo := &testChannelStatsRepo{
		statsByChannelID: map[int64]*models.ChannelStats{
			channelID: {
				ChannelID:   channelID,
				ChannelName: "Existing-Channel",
			},
		},
	}
	task := &StatsAggregationTask{
		channelStatsRepo: repo,
	}
	merged, err := task.mergeChannelStats(context.Background(), &models.ChannelStats{
		ChannelID:   channelID,
		ChannelName: "",
	}, nil)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if merged.ChannelName != "Existing-Channel" {
		t.Fatalf("expected existing channel name to be kept, got %q", merged.ChannelName)
	}
}

func findChannelModelStat(items []*models.ChannelModelStats, channelID int64, model string) *models.ChannelModelStats {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.ChannelID == channelID && item.Model == model {
			return item
		}
	}
	return nil
}

func findTokenStat(items []*models.TokenStats, tokenID int64) *models.TokenStats {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.TokenID == tokenID {
			return item
		}
	}
	return nil
}

func findUserStat(items []*models.UserStats, userID int64) *models.UserStats {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.UserID == userID {
			return item
		}
	}
	return nil
}

func findDailyStat(items []*models.UserUsageDailyStats, userID int64, date string) *models.UserUsageDailyStats {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.UserID == userID && item.Date == date {
			return item
		}
	}
	return nil
}

func findHourlyStat(items []*models.UserUsageHourlyStats, userID int64, date string, hour int) *models.UserUsageHourlyStats {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.UserID == userID && item.Date == date && item.Hour == hour {
			return item
		}
	}
	return nil
}

func TestStatsAggregationTaskRun_EmptyIncrementalLogsStillRefreshesObservationWindow(t *testing.T) {
	now := time.Now()
	channelID := int64(101)
	model := "gpt-4o-mini"

	aggRepo := &testAggregationTaskRepo{
		latestCompleted: &models.AggregationTask{
			TaskName: StatsAggregationTaskName,
			EndID:    200,
			Status:   int8(models.AggregationTaskStatusDone),
		},
	}
	requestLogRepo := &testRequestLogRepo{}
	channelStatsRepo := &testChannelStatsRepo{
		statsByChannelID: map[int64]*models.ChannelStats{
			channelID: {
				ChannelID: channelID,
			},
		},
		windowLogs: []*models.RequestLog{
			{
				ChannelID:     channelID,
				UpstreamModel: model,
				Status:        models.RequestStatusSuccess,
				InputToken:    3,
				OutputToken:   6,
				TransferTime:  500,
				TTFT:          120,
				CreatedAt:     now.Add(-10 * time.Minute),
			},
		},
	}
	channelModelStatsRepo := &testChannelModelStatsRepo{
		statsByKey: map[channelModelKey]*models.ChannelModelStats{
			{
				channelID: channelID,
				model:     model,
			}: {
				ChannelID: channelID,
				Model:     model,
			},
		},
	}
	tx := &testTransaction{}

	task := &StatsAggregationTask{
		logger:                &log.Logger{Logger: zap.NewNop()},
		tm:                    tx,
		aggregationTaskRepo:   aggRepo,
		requestLogRepo:        requestLogRepo,
		channelStatsRepo:      channelStatsRepo,
		channelModelStatsRepo: channelModelStatsRepo,
		nextID: func() int64 {
			return 1
		},
	}

	err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if tx.called == 0 {
		t.Fatal("expected transaction to be used for window refresh when incremental logs are empty")
	}
	if channelStatsRepo.upsertCount == 0 {
		t.Fatal("expected channel stats upsert for window refresh")
	}
	if channelModelStatsRepo.upsertCount == 0 {
		t.Fatal("expected channel-model stats upsert for window refresh")
	}

	updatedChannel := channelStatsRepo.statsByChannelID[channelID]
	if updatedChannel == nil {
		t.Fatal("missing updated channel stats")
	}
	if len(updatedChannel.Window3H.Buckets) != models.ObservationBucketCount {
		t.Fatalf("expected %d window buckets, got %d", models.ObservationBucketCount, len(updatedChannel.Window3H.Buckets))
	}
	if !windowHasTraffic(updatedChannel.Window3H) {
		t.Fatal("expected refreshed channel window to include traffic from recent logs")
	}

	updatedModel := channelModelStatsRepo.statsByKey[channelModelKey{
		channelID: channelID,
		model:     model,
	}]
	if updatedModel == nil {
		t.Fatal("missing updated channel-model stats")
	}
	if len(updatedModel.Window3H.Buckets) != models.ObservationBucketCount {
		t.Fatalf("expected %d model window buckets, got %d", models.ObservationBucketCount, len(updatedModel.Window3H.Buckets))
	}
	if !windowHasTraffic(updatedModel.Window3H) {
		t.Fatal("expected refreshed channel-model window to include traffic from recent logs")
	}
}

func windowHasTraffic(window models.ObservationWindow3H) bool {
	for _, bucket := range window.Buckets {
		if bucket.RequestSuccess > 0 || bucket.RequestFailed > 0 {
			return true
		}
	}
	return false
}

type testTransaction struct {
	called int
}

func (t *testTransaction) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	t.called++
	return fn(ctx)
}

type testAggregationTaskRepo struct {
	latestCompleted *models.AggregationTask
}

func (r *testAggregationTaskRepo) Create(ctx context.Context, task *models.AggregationTask) error {
	return nil
}

func (r *testAggregationTaskRepo) Update(ctx context.Context, task *models.AggregationTask) error {
	return nil
}

func (r *testAggregationTaskRepo) GetByID(ctx context.Context, id int64) (*models.AggregationTask, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *testAggregationTaskRepo) GetLatestByTaskName(ctx context.Context, taskName string) (*models.AggregationTask, error) {
	if r.latestCompleted == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.latestCompleted, nil
}

func (r *testAggregationTaskRepo) GetLatestCompletedByTaskName(ctx context.Context, taskName string) (*models.AggregationTask, error) {
	if r.latestCompleted == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.latestCompleted, nil
}

type testRequestLogRepo struct{}

func (r *testRequestLogRepo) Create(ctx context.Context, log *models.RequestLog) error {
	return nil
}

func (r *testRequestLogRepo) BatchCreate(ctx context.Context, logs []*models.RequestLog) error {
	return nil
}

func (r *testRequestLogRepo) GetByID(ctx context.Context, id int64) (*models.RequestLog, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *testRequestLogRepo) List(ctx context.Context, opt repository.RequestLogListOption) ([]*models.RequestLog, int64, error) {
	return []*models.RequestLog{}, 0, nil
}

func (r *testRequestLogRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func (r *testRequestLogRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return false, nil
}

type testChannelStatsRepo struct {
	statsByChannelID map[int64]*models.ChannelStats
	windowLogs       []*models.RequestLog
	upsertCount      int
}

func (r *testChannelStatsRepo) Upsert(ctx context.Context, stats *models.ChannelStats) error {
	if stats == nil {
		return errors.New("channel stats is nil")
	}
	cloned := *stats
	r.statsByChannelID[stats.ChannelID] = &cloned
	r.upsertCount++
	return nil
}

func (r *testChannelStatsRepo) GetByChannelID(ctx context.Context, channelID int64) (*models.ChannelStats, error) {
	if item, ok := r.statsByChannelID[channelID]; ok {
		cloned := *item
		return &cloned, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *testChannelStatsRepo) ListAll(ctx context.Context) ([]*models.ChannelStats, error) {
	items := make([]*models.ChannelStats, 0, len(r.statsByChannelID))
	for _, item := range r.statsByChannelID {
		cloned := *item
		items = append(items, &cloned)
	}
	return items, nil
}

func (r *testChannelStatsRepo) ListByChannelIDs(ctx context.Context, channelIDs []int64) ([]*models.ChannelStats, error) {
	items := make([]*models.ChannelStats, 0, len(channelIDs))
	for _, id := range channelIDs {
		if item, ok := r.statsByChannelID[id]; ok {
			cloned := *item
			items = append(items, &cloned)
		}
	}
	return items, nil
}

func (r *testChannelStatsRepo) ListRequestLogsByChannelIDsAndRange(ctx context.Context, channelIDs []int64, start, end time.Time) ([]*models.RequestLog, error) {
	return nil, nil
}

func (r *testChannelStatsRepo) ListRequestLogsByRange(ctx context.Context, start, end time.Time) ([]*models.RequestLog, error) {
	return r.windowLogs, nil
}

type testChannelModelStatsRepo struct {
	statsByKey  map[channelModelKey]*models.ChannelModelStats
	upsertCount int
}

func (r *testChannelModelStatsRepo) Upsert(ctx context.Context, stats *models.ChannelModelStats) error {
	if stats == nil {
		return errors.New("channel model stats is nil")
	}
	key := channelModelKey{
		channelID: stats.ChannelID,
		model:     stats.Model,
	}
	cloned := *stats
	r.statsByKey[key] = &cloned
	r.upsertCount++
	return nil
}

func (r *testChannelModelStatsRepo) GetByChannelModel(ctx context.Context, channelID int64, model string) (*models.ChannelModelStats, error) {
	key := channelModelKey{
		channelID: channelID,
		model:     model,
	}
	if item, ok := r.statsByKey[key]; ok {
		cloned := *item
		return &cloned, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *testChannelModelStatsRepo) ListByChannelID(ctx context.Context, channelID int64) ([]*models.ChannelModelStats, error) {
	items := make([]*models.ChannelModelStats, 0)
	for key, item := range r.statsByKey {
		if key.channelID != channelID {
			continue
		}
		cloned := *item
		items = append(items, &cloned)
	}
	return items, nil
}

func (r *testChannelModelStatsRepo) ListRequestLogsByChannelModelAndRange(ctx context.Context, channelID int64, model string, start, end time.Time) ([]*models.RequestLog, error) {
	return nil, nil
}

func (r *testChannelModelStatsRepo) ListRequestLogsByChannelAndRange(ctx context.Context, channelID int64, start, end time.Time) ([]*models.RequestLog, error) {
	return nil, nil
}
