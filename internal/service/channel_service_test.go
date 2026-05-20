package service

import (
	"testing"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/sid"
)

func TestBuildChannelByCreateReq_PopulatesAutoSyncUpstreamModels(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.Sid.Id = 1
	svc := &channelService{
		Service: &Service{
			sid: sid.NewSid(cfg),
		},
	}

	req := v1.CreateChannelRequest{
		Type:    models.ChannelTypeOpenAI,
		Name:    "test-channel",
		BaseURL: "https://example.com",
		APIKey:  "sk-test",
		Settings: &models.ChannelSettings{
			AutoSyncUpstreamModels:  true,
			SyncModelWhitelistRegex: "^gpt-",
			SyncModelBlacklistRegex: "audio",
		},
	}

	channel, err := svc.buildChannelByCreateReq(req)
	if err != nil {
		t.Fatalf("buildChannelByCreateReq failed: %v", err)
	}
	if !channel.Settings.AutoSyncUpstreamModels {
		t.Fatalf("expected autoSyncUpstreamModels=true, got=false")
	}
	if channel.Settings.SyncModelWhitelistRegex != "^gpt-" {
		t.Fatalf("unexpected syncModelWhitelistRegex: %q", channel.Settings.SyncModelWhitelistRegex)
	}
	if channel.Settings.SyncModelBlacklistRegex != "audio" {
		t.Fatalf("unexpected syncModelBlacklistRegex: %q", channel.Settings.SyncModelBlacklistRegex)
	}
}

func TestPatchChannelByUpdateReq_PopulatesAutoSyncUpstreamModels(t *testing.T) {
	channel := &models.Channel{
		Settings: models.ChannelSettings{
			AutoSyncUpstreamModels:  false,
			SyncModelWhitelistRegex: "",
			SyncModelBlacklistRegex: "",
		},
	}

	req := v1.UpdateChannelRequest{
		Settings: &models.ChannelSettings{
			AutoSyncUpstreamModels:  true,
			SyncModelWhitelistRegex: "^gpt-",
			SyncModelBlacklistRegex: "mini$",
		},
	}

	patchChannelByUpdateReq(channel, req)
	if !channel.Settings.AutoSyncUpstreamModels {
		t.Fatalf("expected autoSyncUpstreamModels=true, got=false")
	}
	if channel.Settings.SyncModelWhitelistRegex != "^gpt-" {
		t.Fatalf("unexpected syncModelWhitelistRegex: %q", channel.Settings.SyncModelWhitelistRegex)
	}
	if channel.Settings.SyncModelBlacklistRegex != "mini$" {
		t.Fatalf("unexpected syncModelBlacklistRegex: %q", channel.Settings.SyncModelBlacklistRegex)
	}
}
