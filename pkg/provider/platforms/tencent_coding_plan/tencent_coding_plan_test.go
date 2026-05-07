package tencentcodingplan

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func TestNewClient_DefaultConfig(t *testing.T) {
	c := NewClient(http.DefaultClient, "", "test-key")

	if c.BaseURL != DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", c.BaseURL, DefaultBaseURL)
	}
	if c.Identifier() != PlatformId {
		t.Fatalf("unexpected platform id: got=%q want=%q", c.Identifier(), PlatformId)
	}
	if !c.HasAbility(types.AbilityChatCompletions) {
		t.Fatalf("chat completions ability should be enabled")
	}
	if !c.HasAbility(types.AbilityClaudeMessages) {
		t.Fatalf("claude messages ability should be enabled")
	}
}

func TestClientModels_UsesCodingV3Path(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("X-API-Key")
		_, _ = w.Write([]byte(`{"data":[{"id":"tencent-coding-a"},{"id":"tencent-coding-b"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	if gotPath != "/coding/v3/models" {
		t.Fatalf("unexpected models path: got=%q want=%q", gotPath, "/coding/v3/models")
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("unexpected authorization header: got=%q", gotAuth)
	}
	if gotAPIKey != "test-key" {
		t.Fatalf("unexpected x-api-key header: got=%q", gotAPIKey)
	}

	want := []string{"tencent-coding-a", "tencent-coding-b"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected models: got=%v want=%v", models, want)
	}
}
