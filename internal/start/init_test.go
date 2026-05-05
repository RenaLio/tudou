package start

import (
	"testing"

	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
)

func TestReplayRequestLogsToRegistry(t *testing.T) {
	reg := loadbalancer.NewRegistry()

	ch := &models.Channel{
		ID:    1,
		Type:  "openai",
		Name:  "ch-1",
		Model: "gpt-4o",
	}
	reg.ReloadChannel(ch)

	logs := []*models.RequestLog{
		{
			ChannelID:     1,
			Model:         "gpt-4o",
			Status:        models.RequestStatusSuccess,
			OutputToken:   200,
			TransferTime:  2000,
			TTFT:          1000,
			ErrorCode:     "200",
			UpstreamModel: "gpt-4o",
			Extra: models.RequestExtra{
				RetryTrace: []models.RetryDetail{
					{
						ChannelID:     1,
						ChannelName:   "ch-1",
						UpstreamModel: "gpt-4o",
						StatusCode:    503,
						StatusBody:    "upstream timeout",
					},
				},
			},
		},
		{
			ChannelID:     1,
			Model:         "gpt-4o",
			Status:        models.RequestStatusFail,
			OutputToken:   0,
			TransferTime:  1500,
			TTFT:          500,
			ErrorCode:     "500",
			UpstreamModel: "gpt-4o",
		},
		// should be ignored: validation error
		{
			ChannelID:    1,
			Model:        "gpt-4o",
			Status:       models.RequestStatusFail,
			ErrorCode:    "400",
			TransferTime: 1000,
		},
		// should be ignored: invalid channel
		{
			ChannelID: 999,
			Model:     "gpt-4o",
			Status:    models.RequestStatusSuccess,
			ErrorCode: "200",
		},
		// should be ignored: invalid model
		{
			ChannelID: 1,
			Model:     "",
			Status:    models.RequestStatusSuccess,
			ErrorCode: "200",
		},
	}

	replayRequestLogsToRegistry(reg, logs)

	ep := reg.GetEndpoint("gpt-4o", 1)
	if ep == nil {
		t.Fatalf("endpoint should exist")
	}

	// Initial TTFT is 1600.
	// After success(ttft=1000): 0.8*1600 + 0.2*1000 = 1480
	// After fail(ttft ignored): unchanged 1480
	if ep.EmaTTFT != 1480 {
		t.Fatalf("unexpected ema ttft, got=%v want=1480", ep.EmaTTFT)
	}

	// Initial TPS is 100.
	// After success(tps=100): unchanged 100
	// After fail(tps ignored): unchanged 100
	if ep.EmaTPS != 100 {
		t.Fatalf("unexpected ema tps, got=%v want=100", ep.EmaTPS)
	}

	// Initial success rate is 1.0
	// After success(main log): 1.0
	// After fail(retryTrace flatten log): 0.95
	// After fail(main fail log): 0.9025
	if ep.EmaSuccessRate != 0.9025 {
		t.Fatalf("unexpected ema success rate, got=%v want=0.9025", ep.EmaSuccessRate)
	}

	chState := reg.GetChannelById(1)
	if chState == nil {
		t.Fatalf("channel should exist")
	}

	// Initial channel success rate is 0.
	// After success(main log): 0.05
	// After fail(retryTrace flatten log): 0.0475
	// After fail(main fail log): 0.045125
	if chState.SuccessRate != 0.045125 {
		t.Fatalf("unexpected channel success rate, got=%v want=0.045125", chState.SuccessRate)
	}
}
