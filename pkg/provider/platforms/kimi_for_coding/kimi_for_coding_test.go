package kimiforcoding

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientModels_ReturnsUnion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"kimi-k2.5"},{"id":"remote-extra"},{"id":"kimi-k2.6"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	wantPrefix := []string{
		"kimi-k2.6",
		"kimi-for-coding",
		"kimi-k2.5",
		"kimi-k2-thinking",
	}
	if len(models) != len(wantPrefix)+1 {
		t.Fatalf("unexpected models length: got=%d want=%d (%v)", len(models), len(wantPrefix)+1, models)
	}
	for i, want := range wantPrefix {
		if models[i] != want {
			t.Fatalf("unexpected model at %d: got=%s want=%s", i, models[i], want)
		}
	}
	if models[len(models)-1] != "remote-extra" {
		t.Fatalf("expected remote-only model appended, got=%v", models)
	}
}

func TestClientModels_FetchErrorReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "bad-key")
	_, err := c.Models()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
