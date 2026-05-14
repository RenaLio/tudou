package tasks

import (
	"context"
	"errors"
	"slices"
	"testing"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"go.uber.org/zap"
)

func TestChannelModelSyncTask_Run_UpdatesEnabledChannels(t *testing.T) {
	channelSvc := &testChannelSyncChannelService{
		pages: [][]v1.ChannelResponse{
			{
				{
					ID:      1,
					Type:    models.ChannelTypeOpenAI,
					Name:    "ch-1",
					BaseURL: "https://example.com",
					APIKey:  "k1",
					Model:   "a",
					Settings: models.ChannelSettings{
						AutoSyncUpstreamModels: true,
					},
				},
				{
					ID:      2,
					Type:    models.ChannelTypeOpenAI,
					Name:    "ch-2",
					BaseURL: "https://example.com",
					APIKey:  "k2",
					Model:   "b",
					Settings: models.ChannelSettings{
						AutoSyncUpstreamModels: false,
					},
				},
			},
		},
	}
	fetcher := &testChannelModelFetcher{
		resp: map[int64][]string{
			1: {"gpt-4o", "gpt-4.1-mini", "gpt-4o"},
			2: {"should-not-call"},
		},
	}
	task := NewChannelModelSyncTask(
		&log.Logger{Logger: zap.NewNop()},
		channelSvc,
		fetcher,
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(channelSvc.updated) != 1 {
		t.Fatalf("expected 1 updated channel, got=%d", len(channelSvc.updated))
	}
	if channelSvc.updated[0].id != 1 {
		t.Fatalf("unexpected updated channel id: %d", channelSvc.updated[0].id)
	}
	if channelSvc.updated[0].model != "gpt-4.1-mini,gpt-4o" {
		t.Fatalf("unexpected normalized model: %q", channelSvc.updated[0].model)
	}
	if !slices.Equal(fetcher.calledIDs, []int64{1}) {
		t.Fatalf("unexpected fetch calls: %+v", fetcher.calledIDs)
	}

	statsAny, err := task.CurrentStats()
	if err != nil {
		t.Fatalf("CurrentStats failed: %v", err)
	}
	stats := statsAny.(ChannelModelSyncTaskStats)
	if stats.TotalChannels != 2 || stats.SyncEnabled != 1 || stats.UpdatedChannels != 1 || stats.FailedChannels != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestChannelModelSyncTask_Run_FetchErrorContinues(t *testing.T) {
	channelSvc := &testChannelSyncChannelService{
		pages: [][]v1.ChannelResponse{
			{
				{
					ID:      1,
					Type:    models.ChannelTypeOpenAI,
					BaseURL: "https://example.com",
					APIKey:  "k1",
					Settings: models.ChannelSettings{
						AutoSyncUpstreamModels: true,
					},
				},
				{
					ID:      2,
					Type:    models.ChannelTypeOpenAI,
					BaseURL: "https://example.com",
					APIKey:  "k2",
					Model:   "old",
					Settings: models.ChannelSettings{
						AutoSyncUpstreamModels: true,
					},
				},
			},
		},
	}
	fetcher := &testChannelModelFetcher{
		resp: map[int64][]string{
			2: {"x"},
		},
		errFor: map[int64]error{
			1: errors.New("fetch failed"),
		},
	}
	task := NewChannelModelSyncTask(
		&log.Logger{Logger: zap.NewNop()},
		channelSvc,
		fetcher,
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run should not fail on per-channel fetch error, got=%v", err)
	}
	if len(channelSvc.updated) != 1 || channelSvc.updated[0].id != 2 {
		t.Fatalf("unexpected updates: %+v", channelSvc.updated)
	}

	statsAny, err := task.CurrentStats()
	if err != nil {
		t.Fatalf("CurrentStats failed: %v", err)
	}
	stats := statsAny.(ChannelModelSyncTaskStats)
	if stats.FailedChannels != 1 || stats.UpdatedChannels != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestNormalizeModelList(t *testing.T) {
	got := normalizeModelList([]string{" b ", "a", "", "a"})
	if got != "a,b" {
		t.Fatalf("unexpected normalized value: %q", got)
	}
	if normalizeModelList(nil) != "" {
		t.Fatalf("expected empty result for nil input")
	}
}

type testChannelSyncChannelService struct {
	pages     [][]v1.ChannelResponse
	listCalls int
	updated   []testChannelSyncUpdate
}

type testChannelSyncUpdate struct {
	id    int64
	model string
}

func (s *testChannelSyncChannelService) List(_ context.Context, _ v1.ListChannelsRequest) (*v1.ListResponse[v1.ChannelResponse], error) {
	if s.listCalls >= len(s.pages) {
		return &v1.ListResponse[v1.ChannelResponse]{Items: []v1.ChannelResponse{}}, nil
	}
	items := s.pages[s.listCalls]
	s.listCalls++
	return &v1.ListResponse[v1.ChannelResponse]{
		Items:    items,
		Total:    int64(len(items)),
		Page:     int64(s.listCalls),
		PageSize: int64(len(items)),
	}, nil
}

func (s *testChannelSyncChannelService) Update(_ context.Context, id int64, req v1.UpdateChannelRequest) (*v1.ChannelResponse, error) {
	model := ""
	if req.Model != nil {
		model = *req.Model
	}
	s.updated = append(s.updated, testChannelSyncUpdate{
		id:    id,
		model: model,
	})
	return &v1.ChannelResponse{ID: id, Model: model}, nil
}

type testChannelModelFetcher struct {
	resp      map[int64][]string
	errFor    map[int64]error
	calledIDs []int64
}

func (f *testChannelModelFetcher) FetchModel(_ context.Context, req *v1.FetchModelRequest) ([]string, error) {
	id := parseKeyID(req.APIKey)
	f.calledIDs = append(f.calledIDs, id)
	if err := f.errFor[id]; err != nil {
		return nil, err
	}
	return f.resp[id], nil
}

func parseKeyID(key string) int64 {
	if len(key) < 2 {
		return 0
	}
	switch key {
	case "k1":
		return 1
	case "k2":
		return 2
	default:
		return 0
	}
}
