package scnetcoding

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func TestNewClient_DefaultBaseURL(t *testing.T) {
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

func TestNewClient_CustomBaseURL(t *testing.T) {
	custom := "https://custom.example.com/api"
	c := NewClient(http.DefaultClient, custom, "test-key")

	if c.BaseURL != custom {
		t.Fatalf("unexpected base url: got=%q want=%q", c.BaseURL, custom)
	}
}

func TestModels_IntersectionWithRemote(t *testing.T) {
	// Remote returns MiniMax-M2.5, Qwen3-235B-A22B, and a remote-only model.
	// Only the two local models should be returned (intersection).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"MiniMax-M2.5"},{"id":"Qwen3-235B-A22B"},{"id":"remote-only-model"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	want := []string{"MiniMax-M2.5", "Qwen3-235B-A22B"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected models: got=%v want=%v", models, want)
	}
}

func TestModels_PartialIntersection(t *testing.T) {
	// Remote only returns MiniMax-M2.5; Qwen3-235B-A22B is not remotely available.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"MiniMax-M2.5"},{"id":"other-model"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	want := []string{"MiniMax-M2.5"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected models: got=%v want=%v", models, want)
	}
}

func TestModels_EmptyIntersection_FallbackToLocal(t *testing.T) {
	// Remote returns no overlapping models → fallback to local list.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"unrelated-a"},{"id":"unrelated-b"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	if !reflect.DeepEqual(models, LocalModelList) {
		t.Fatalf("expected fallback to local list: got=%v want=%v", models, LocalModelList)
	}
}

func TestModels_RemoteError_FallbackToLocal(t *testing.T) {
	// Remote returns 500 → fallback to local list (no error propagated).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() should not return error on remote failure, got=%v", err)
	}

	if !reflect.DeepEqual(models, LocalModelList) {
		t.Fatalf("expected fallback to local list: got=%v want=%v", models, LocalModelList)
	}
}

func TestModels_RemoteDeduplication(t *testing.T) {
	// Remote returns duplicates; intersection should still work correctly.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"MiniMax-M2.5"},{"id":"MiniMax-M2.5"},{"id":"Qwen3-235B-A22B"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	want := []string{"MiniMax-M2.5", "Qwen3-235B-A22B"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected models: got=%v want=%v", models, want)
	}
}

func TestModels_AuthHeaders(t *testing.T) {
	var gotAuth, gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("X-API-Key")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "my-secret-key")
	_, _ = c.Models()

	if gotAuth != "Bearer my-secret-key" {
		t.Fatalf("unexpected Authorization: got=%q", gotAuth)
	}
	if gotAPIKey != "my-secret-key" {
		t.Fatalf("unexpected X-API-Key: got=%q", gotAPIKey)
	}
}

func TestModels_RemoteRequestPath(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	_, _ = c.Models()

	if gotPath != "/v1/models" {
		t.Fatalf("unexpected models request path: got=%q want=%q", gotPath, "/v1/models")
	}
}
