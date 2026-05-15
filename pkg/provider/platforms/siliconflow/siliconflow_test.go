package siliconflow

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient(http.DefaultClient, "", "test-key")

	if c.Identifier() != PlatformId {
		t.Fatalf("unexpected platform id: got=%q want=%q", c.Identifier(), PlatformId)
	}
	if c.BaseURL != DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", c.BaseURL, DefaultBaseURL)
	}
	if !c.HasAbility(types.AbilityChatCompletions) {
		t.Fatalf("chat completions ability should be enabled")
	}
	if !c.HasAbility(types.AbilityClaudeMessages) {
		t.Fatalf("claude messages ability should be enabled")
	}
	if !c.HasAbility(types.AbilityEmbeddings) {
		t.Fatalf("embeddings ability should be enabled")
	}
}

func TestFormatPathMap(t *testing.T) {
	if got := DefaultFormatPathMap[types.FormatChatCompletion]; got != "/v1/chat/completions" {
		t.Fatalf("unexpected chat completions path: got=%q", got)
	}
	if got := DefaultFormatPathMap[types.FormatClaudeMessages]; got != "/v1/messages" {
		t.Fatalf("unexpected claude messages path: got=%q", got)
	}
	if got := DefaultFormatPathMap[types.FormatOpenAIEmbeddings]; got != "/v1/embeddings" {
		t.Fatalf("unexpected embeddings path: got=%q", got)
	}
}

func TestModels_FetchesV1Models(t *testing.T) {
	var gotPath string
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[{"id":"Qwen/Qwen3-8B"},{"id":"deepseek-ai/DeepSeek-V3"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	if gotPath != "/v1/models" {
		t.Fatalf("unexpected models path: got=%q want=%q", gotPath, "/v1/models")
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("unexpected Authorization header: got=%q", gotAuth)
	}

	want := []string{"Qwen/Qwen3-8B", "deepseek-ai/DeepSeek-V3"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected models: got=%v want=%v", models, want)
	}
}
