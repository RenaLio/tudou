package service

import (
	"context"
	"testing"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type testChannelGroupRepo struct {
	group *models.ChannelGroup
	err   error
}

func (r *testChannelGroupRepo) Create(context.Context, *models.ChannelGroup) error {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) BatchCreate(context.Context, []*models.ChannelGroup) error {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) GetByID(context.Context, int64) (*models.ChannelGroup, error) {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) GetByIDWithChannels(context.Context, int64) (*models.ChannelGroup, error) {
	return r.group, r.err
}

func (r *testChannelGroupRepo) GetByName(context.Context, string) (*models.ChannelGroup, error) {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) List(context.Context, repository.ChannelGroupListOption) ([]*models.ChannelGroup, int64, error) {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) Update(context.Context, *models.ChannelGroup) error {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) Delete(context.Context, int64) error {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) ReplaceChannels(context.Context, int64, []int64) error {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) Exists(context.Context, int64) (bool, error) {
	panic("unexpected call")
}

func (r *testChannelGroupRepo) PreLoadRegistryData(context.Context) ([]*models.ChannelGroup, error) {
	panic("unexpected call")
}

var _ repository.ChannelGroupRepo = (*testChannelGroupRepo)(nil)

func TestRelayServiceGetTokenModels_FiltersUnavailableChannels(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Hour)
	available := now.Add(time.Hour)

	svc := &RelayService{
		groupRepo: &testChannelGroupRepo{
			group: &models.ChannelGroup{
				Channels: []models.Channel{
					{
						ID:        1,
						Type:      models.ChannelTypeOpenAI,
						Status:    models.ChannelStatusEnabled,
						Model:     "gpt-4o-mini,shared-model",
						ExpiredAt: &available,
						CreatedAt: now,
					},
					{
						ID:        2,
						Type:      models.ChannelTypeClaude,
						Status:    models.ChannelStatusDisabled,
						Model:     "disabled-model,shared-model",
						ExpiredAt: &available,
						CreatedAt: now,
					},
					{
						ID:        3,
						Type:      models.ChannelTypeClaude,
						Status:    models.ChannelStatusEnabled,
						Model:     "expired-model",
						ExpiredAt: &expired,
						CreatedAt: now,
					},
				},
			},
		},
	}

	resp, err := svc.GetTokenModels(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetTokenModels returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Object != "list" {
		t.Fatalf("unexpected object: got=%q want=%q", resp.Object, "list")
	}

	got := make(map[string]v1.RelayModelItemResp, len(resp.Data))
	for _, item := range resp.Data {
		got[item.Id] = item
	}

	if len(got) != 2 {
		t.Fatalf("unexpected model count: got=%d want=%d", len(got), 2)
	}
	if _, ok := got["gpt-4o-mini"]; !ok {
		t.Fatalf("expected available model gpt-4o-mini to be returned")
	}
	if _, ok := got["shared-model"]; !ok {
		t.Fatalf("expected shared-model from available channel to be returned")
	}
	if _, ok := got["disabled-model"]; ok {
		t.Fatalf("did not expect disabled-model from disabled channel to be returned")
	}
	if _, ok := got["expired-model"]; ok {
		t.Fatalf("did not expect expired-model from expired channel to be returned")
	}
}
