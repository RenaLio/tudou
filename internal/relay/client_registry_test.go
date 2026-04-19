package relay_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/relay"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func newChannel(id int64, updated time.Time) *models.Channel {
	return &models.Channel{
		ID:        id,
		Name:      "test",
		BaseURL:   "https://api.example.com",
		APIKey:    "sk-test",
		Type:      models.ChannelTypeOpenAI,
		UpdatedAt: updated,
	}
}

func TestClientRegistry_GetCreatesAndCaches(t *testing.T) {
	reg := relay.NewClientRegistry(&http.Client{Timeout: 1 * time.Second})

	ch := newChannel(1, time.Now())
	abilities := []types.Ability{types.AbilityChat}
	c1 := reg.Get(ch, abilities)
	if c1 == nil {
		t.Fatal("expected non-nil client")
	}
	c2 := reg.Get(ch, abilities)
	if c1 != c2 {
		t.Fatal("expected cached client to be reused")
	}
}

func TestClientRegistry_InvalidateOnUpdatedAtChange(t *testing.T) {
	reg := relay.NewClientRegistry(&http.Client{Timeout: 1 * time.Second})

	t1 := time.Now()
	t2 := t1.Add(1 * time.Second)

	ch1 := newChannel(1, t1)
	ch2 := newChannel(1, t2)
	ch2.APIKey = "sk-rotated"

	abilities := []types.Ability{types.AbilityChat}
	c1 := reg.Get(ch1, abilities)
	c2 := reg.Get(ch2, abilities)
	if c1 == c2 {
		t.Fatal("expected a new client after UpdatedAt change")
	}
}

func TestClientRegistry_InvalidateExplicit(t *testing.T) {
	reg := relay.NewClientRegistry(&http.Client{Timeout: 1 * time.Second})

	ch := newChannel(1, time.Now())
	abilities := []types.Ability{types.AbilityChat}
	c1 := reg.Get(ch, abilities)
	reg.Invalidate(1)
	c2 := reg.Get(ch, abilities)
	if c1 == c2 {
		t.Fatal("expected a new client after explicit invalidate")
	}
}

func TestClientRegistry_NilChannel(t *testing.T) {
	reg := relay.NewClientRegistry(&http.Client{Timeout: 1 * time.Second})
	if got := reg.Get(nil, nil); got != nil {
		t.Fatalf("expected nil for nil channel, got %v", got)
	}
}
