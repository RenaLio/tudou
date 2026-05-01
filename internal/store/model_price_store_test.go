package store

import (
	"testing"
	"time"
)

func TestFindSimilarPath_PrefersPricedPath(t *testing.T) {
	s := &ModelPriceStore{
		RawData: []byte(`{
			"openrouter": {
				"models": {
					"qwen/qwen3-30b-a3b": {
						"cost": {"input": 0.5, "output": 1}
					},
					"qwen/qwen3-30b-a3b:free": {
						"cost": {"input": 0, "output": 0}
					}
				}
			}
		}`),
		Models: []string{
			"openrouter#qwen/qwen3-30b-a3b",
			"openrouter#qwen/qwen3-30b-a3b:free",
		},
		LatestFetchTime: time.Now(),
	}

	got := s.FindSimilarPath("openrouter", "qwen/qwen3-30b-a3b:free")
	want := "openrouter.models.qwen/qwen3-30b-a3b"
	if got != want {
		t.Fatalf("expected priced path %q, got %q", want, got)
	}
}

func TestFindSimilarPath_FallbackToFreeWhenNoPricedPath(t *testing.T) {
	s := &ModelPriceStore{
		RawData: []byte(`{
			"openrouter": {
				"models": {
					"qwen/qwen3-30b-a3b:free": {
						"cost": {"input": 0, "output": 0}
					}
				}
			}
		}`),
		Models: []string{
			"openrouter#qwen/qwen3-30b-a3b:free",
		},
		LatestFetchTime: time.Now(),
	}

	got := s.FindSimilarPath("openrouter", "qwen/qwen3-30b-a3b:free")
	want := "openrouter.models.qwen/qwen3-30b-a3b:free"
	if got != want {
		t.Fatalf("expected fallback path %q, got %q", want, got)
	}
}
