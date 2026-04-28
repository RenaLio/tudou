package tasks

import (
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
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
