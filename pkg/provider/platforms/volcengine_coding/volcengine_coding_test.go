package volcenginecoding

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestModels_ReturnsLocalFirstThenRemoteOnly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/models" {
			t.Fatalf("unexpected path: got=%q want=%q", r.URL.Path, "/v3/models")
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"glm-5.1"},{"id":"remote-extra"},{"id":"minimax-m2.7"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	want := append(append([]string(nil), LocalModelList...), "remote-extra")
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected merged models: got=%v want=%v", models, want)
	}
}

func TestModels_RemoteFailureFallsBackToLocalList(t *testing.T) {
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
