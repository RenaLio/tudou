package modelcatalog

import (
	"reflect"
	"testing"
)

func TestLoadReturnsCopy(t *testing.T) {
	models, err := Load("alibaba-coding-plan-cn")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(models) == 0 {
		t.Fatalf("expected non-empty models list")
	}

	originalFirst := models[0]
	models[0] = "mutated"

	reloaded, err := Load("alibaba-coding-plan-cn")
	if err != nil {
		t.Fatalf("Load() reload error = %v", err)
	}
	if len(reloaded) == 0 {
		t.Fatalf("expected non-empty reloaded models list")
	}
	if reloaded[0] != originalFirst {
		t.Fatalf("expected load to return a defensive copy: got=%q want=%q", reloaded[0], originalFirst)
	}
}

func TestLoadUnknownPlatform(t *testing.T) {
	if _, err := Load("missing-platform"); err == nil {
		t.Fatalf("expected missing platform error, got nil")
	}
}

func TestMustLoadUnknownPlatformPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic for missing platform")
		}
	}()
	_ = MustLoad("missing-platform")
}

func TestEmbeddedCatalogFilesAreLoadable(t *testing.T) {
	loaded, err := loadCatalogs()
	if err != nil {
		t.Fatalf("loadCatalogs() error = %v", err)
	}
	if len(loaded) == 0 {
		t.Fatalf("expected embedded model catalogs")
	}

	for platformID, want := range loaded {
		models, loadErr := Load(platformID)
		if loadErr != nil {
			t.Fatalf("Load(%q) error = %v", platformID, loadErr)
		}
		if len(models) == 0 {
			t.Fatalf("expected non-empty models list for %q", platformID)
		}
		if !reflect.DeepEqual(models, want) {
			t.Fatalf("unexpected models for %q: got=%v want=%v", platformID, models, want)
		}
	}
}
