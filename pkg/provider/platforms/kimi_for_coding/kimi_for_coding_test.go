package kimiforcoding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func TestClientModels_ReturnsUnion(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotUA = r.Header.Get("User-Agent")
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
	if gotUA != defaultCLIUserAgent {
		t.Fatalf("unexpected models request UA: got=%q want=%q", gotUA, defaultCLIUserAgent)
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

func TestClientExecute_UserAgentFallback(t *testing.T) {
	t.Run("empty user-agent uses cli ua", func(t *testing.T) {
		got := runExecuteAndCaptureUA(t, "")
		if got != defaultCLIUserAgent {
			t.Fatalf("unexpected UA: got=%q want=%q", got, defaultCLIUserAgent)
		}
	})

	t.Run("browser user-agent uses cli ua", func(t *testing.T) {
		got := runExecuteAndCaptureUA(t, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
		if got != defaultCLIUserAgent {
			t.Fatalf("unexpected UA: got=%q want=%q", got, defaultCLIUserAgent)
		}
	})

	t.Run("non-browser user-agent keeps original", func(t *testing.T) {
		want := "curl/8.7.1"
		got := runExecuteAndCaptureUA(t, want)
		if got != want {
			t.Fatalf("unexpected UA: got=%q want=%q", got, want)
		}
	})
}

func runExecuteAndCaptureUA(t *testing.T, incomingUA string) string {
	t.Helper()

	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","choices":[],"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")

	headers := http.Header{}
	if incomingUA != "" {
		headers.Set("User-Agent", incomingUA)
	}

	_, err := c.Execute(context.Background(), &types.Request{
		Model:   "kimi-for-coding",
		Format:  types.FormatChatCompletion,
		Headers: headers,
		Payload: []byte(`{"model":"kimi-for-coding","messages":[{"role":"user","content":"hello"}]}`),
	}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	return gotUA
}
