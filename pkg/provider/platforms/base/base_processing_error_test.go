package base

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func TestClientExecute_429FillsProcessingErrorForCallback(t *testing.T) {
	const errBody = `{"error":{"message":"rate limit exceeded","code":429}}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(errBody))
	}))
	defer ts.Close()

	client := NewClient(http.DefaultClient, ts.URL, "test-key", "test-provider", []types.Ability{types.AbilityChatCompletions}, nil)

	var captured *types.ResponseMetrics
	resp, err := client.Execute(context.Background(), &types.Request{
		Model:    "test-model",
		Format:   types.FormatChatCompletion,
		IsStream: true, // simulate stream request that gets non-stream 429 body
		Payload:  []byte(`{"model":"test-model","messages":[{"role":"user","content":"hello"}],"stream":true}`),
	}, func(m *types.ResponseMetrics) {
		copied := *m
		captured = &copied
	})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected status: got=%d want=%d", resp.StatusCode, http.StatusTooManyRequests)
	}
	if resp.RequestPath != "/v1/chat/completions" {
		t.Fatalf("unexpected response request path: got=%q want=%q", resp.RequestPath, "/v1/chat/completions")
	}
	if captured == nil {
		t.Fatal("expected callback metrics")
	}
	if captured.ProcessingError == nil {
		t.Fatal("expected ProcessingError to be populated")
	}
	if !strings.Contains(captured.ProcessingError.Error(), "rate limit exceeded") {
		t.Fatalf("unexpected ProcessingError: %q", captured.ProcessingError.Error())
	}
	if captured.RequestPath != "/v1/chat/completions" {
		t.Fatalf("unexpected RequestPath: got=%q want=%q", captured.RequestPath, "/v1/chat/completions")
	}
}
