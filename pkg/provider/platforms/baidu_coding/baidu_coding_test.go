package baiducoding

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestModels_PreservesSupportedOrderWhenFilteringRemote(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/coding/models" {
			t.Fatalf("unexpected path: got=%q want=%q", r.URL.Path, "/v2/coding/models")
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"glm-5"},{"id":"remote-only"},{"id":"kimi-k2.5"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	want := []string{"kimi-k2.5", "glm-5"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected filtered models: got=%v want=%v", models, want)
	}
}

func TestModels_EmptyIntersectionFallsBackToSupportedList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"remote-only-a"},{"id":"remote-only-b"}]}`))
	}))
	defer ts.Close()

	c := NewClient(http.DefaultClient, ts.URL, "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	if !reflect.DeepEqual(models, SupportedModelList) {
		t.Fatalf("expected fallback to supported list: got=%v want=%v", models, SupportedModelList)
	}
}

func TestModels_RemoteFailureFallsBackToSupportedList(t *testing.T) {
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

	if !reflect.DeepEqual(models, SupportedModelList) {
		t.Fatalf("expected fallback to supported list: got=%v want=%v", models, SupportedModelList)
	}
}
