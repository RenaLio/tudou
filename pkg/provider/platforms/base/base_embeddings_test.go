package base

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func TestClientExecute_EmbeddingsFormat(t *testing.T) {
	var gotPath string
	var gotAuth string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[],"usage":{"prompt_tokens":1,"total_tokens":1}}`))
	}))
	defer ts.Close()

	client := NewClient(http.DefaultClient, ts.URL, "test-key", "test-provider", []types.Ability{types.AbilityEmbeddings}, nil)

	resp, err := client.Execute(context.Background(), &types.Request{
		Model:   "text-embedding-3-small",
		Format:  types.FormatOpenAIEmbeddings,
		Payload: []byte(`{"model":"text-embedding-3-small","input":"hello"}`),
	}, nil)
	if err != nil {
		t.Fatalf("execute embeddings returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if gotPath != "/v1/embeddings" {
		t.Fatalf("unexpected request path: %s", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("expected bearer authorization header, got: %q", gotAuth)
	}
}
