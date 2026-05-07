package cucloudcoding

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientModels_UsesV1ModelsEndpoint(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"cu-model-a"},{"id":"cu-model-b"}]}`))
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
	if len(models) != 2 {
		t.Fatalf("unexpected models length: got=%d want=2", len(models))
	}
	if models[0] != "cu-model-a" || models[1] != "cu-model-b" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

