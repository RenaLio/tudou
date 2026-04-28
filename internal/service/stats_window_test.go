package service

import (
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

func TestBuildObservationWindow3H_AlignedAndAggregate(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 7, 0, 0, time.UTC)
	logs := []*models.RequestLog{
		{
			CreatedAt:                 time.Date(2026, 4, 27, 7, 15, 0, 0, time.UTC), // bucket 0 (included)
			InputToken:                10,
			OutputToken:               20,
			CachedCreationInputTokens: 2,
			CachedReadInputTokens:     1,
			Status:                    models.RequestStatusSuccess,
			CostMicros:                100,
			TTFT:                      100,
			TransferTime:              2000,
		},
		{
			CreatedAt:                 time.Date(2026, 4, 27, 7, 29, 59, 0, time.UTC), // bucket 0 (included)
			InputToken:                15,
			OutputToken:               30,
			CachedCreationInputTokens: 1,
			CachedReadInputTokens:     3,
			Status:                    models.RequestStatusFail,
			CostMicros:                200,
			TTFT:                      0, // should be excluded from avgTTFT
			TransferTime:              3000,
		},
		{
			CreatedAt:                 time.Date(2026, 4, 27, 10, 14, 59, 0, time.UTC), // bucket 11 (included)
			InputToken:                5,
			OutputToken:               10,
			CachedCreationInputTokens: 0,
			CachedReadInputTokens:     0,
			Status:                    models.RequestStatusSuccess,
			CostMicros:                50,
			TTFT:                      200,
			TransferTime:              1000,
		},
		{
			CreatedAt:    time.Date(2026, 4, 27, 10, 15, 0, 0, time.UTC), // window end (excluded)
			InputToken:   999,
			OutputToken:  999,
			Status:       models.RequestStatusSuccess,
			CostMicros:   999,
			TTFT:         999,
			TransferTime: 999,
		},
	}

	window := buildObservationWindow3H(now, logs)

	if window.WindowMinutes != 180 {
		t.Fatalf("expected WindowMinutes=180, got %d", window.WindowMinutes)
	}
	if window.BucketMinutes != 15 {
		t.Fatalf("expected BucketMinutes=15, got %d", window.BucketMinutes)
	}
	if len(window.Buckets) != 12 {
		t.Fatalf("expected 12 buckets, got %d", len(window.Buckets))
	}

	first := window.Buckets[0]
	if !first.StartAt.Equal(time.Date(2026, 4, 27, 7, 15, 0, 0, time.UTC)) {
		t.Fatalf("unexpected first bucket start: %s", first.StartAt)
	}
	if !first.EndAt.Equal(time.Date(2026, 4, 27, 7, 30, 0, 0, time.UTC)) {
		t.Fatalf("unexpected first bucket end: %s", first.EndAt)
	}
	if first.InputToken != 25 || first.OutputToken != 50 {
		t.Fatalf("unexpected first bucket token stats: in=%d out=%d", first.InputToken, first.OutputToken)
	}
	if first.CachedCreationInputTokens != 3 || first.CachedReadInputTokens != 4 {
		t.Fatalf("unexpected first bucket cached token stats: create=%d read=%d", first.CachedCreationInputTokens, first.CachedReadInputTokens)
	}
	if first.RequestSuccess != 1 || first.RequestFailed != 1 {
		t.Fatalf("unexpected first bucket req stats: success=%d failed=%d", first.RequestSuccess, first.RequestFailed)
	}
	if first.TotalCostMicros != 300 {
		t.Fatalf("unexpected first bucket cost: %d", first.TotalCostMicros)
	}
	if first.AvgTTFT != 100 {
		t.Fatalf("unexpected first bucket AvgTTFT: %d", first.AvgTTFT)
	}
	if first.AvgTPS != 10 {
		t.Fatalf("unexpected first bucket AvgTPS: %f", first.AvgTPS)
	}

	last := window.Buckets[11]
	if last.InputToken != 5 || last.OutputToken != 10 {
		t.Fatalf("unexpected last bucket token stats: in=%d out=%d", last.InputToken, last.OutputToken)
	}
	if last.RequestSuccess != 1 || last.RequestFailed != 0 {
		t.Fatalf("unexpected last bucket req stats: success=%d failed=%d", last.RequestSuccess, last.RequestFailed)
	}
	if last.AvgTTFT != 200 {
		t.Fatalf("unexpected last bucket AvgTTFT: %d", last.AvgTTFT)
	}
	if last.AvgTPS != 10 {
		t.Fatalf("unexpected last bucket AvgTPS: %f", last.AvgTPS)
	}
}

func TestBuildObservationWindow3H_AlwaysReturns12Buckets(t *testing.T) {
	now := time.Date(2026, 4, 27, 18, 7, 0, 0, time.FixedZone("CST", 8*3600))

	window := buildObservationWindow3H(now, nil)

	if len(window.Buckets) != 12 {
		t.Fatalf("expected 12 buckets, got %d", len(window.Buckets))
	}
	if !window.Buckets[0].StartAt.Equal(time.Date(2026, 4, 27, 7, 15, 0, 0, time.UTC)) {
		t.Fatalf("unexpected first bucket start in  alignment: %s", window.Buckets[0].StartAt)
	}
	if !window.Buckets[11].EndAt.Equal(time.Date(2026, 4, 27, 10, 15, 0, 0, time.UTC)) {
		t.Fatalf("unexpected last bucket end in  alignment: %s", window.Buckets[11].EndAt)
	}
	for i := range window.Buckets {
		b := window.Buckets[i]
		if b.InputToken != 0 || b.OutputToken != 0 || b.RequestSuccess != 0 || b.RequestFailed != 0 || b.TotalCostMicros != 0 || b.AvgTTFT != 0 || b.AvgTPS != 0 {
			t.Fatalf("expected bucket %d to be zero, got %+v", i, b)
		}
	}
}
