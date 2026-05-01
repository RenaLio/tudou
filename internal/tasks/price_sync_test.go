package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/store"
	"go.uber.org/zap"
)

func TestPriceSyncTask_Run_SyncsEnabledModels(t *testing.T) {
	repo := &testPriceSyncModelRepo{
		models: []*models.AIModel{
			{
				ID:   1,
				Name: "gpt-4o",
				Extra: models.AIModelExtra{
					DisableSync:       false,
					SyncModelInfoPath: "  openai.models.gpt-4o  ",
				},
			},
			{
				ID:   2,
				Name: "gpt-4.1-mini",
				Extra: models.AIModelExtra{
					DisableSync:       false,
					SyncModelInfoPath: "",
				},
			},
			{
				ID:   3,
				Name: "gpt-disabled",
				Extra: models.AIModelExtra{
					DisableSync:       true,
					SyncModelInfoPath: "openai.models.gpt-4o",
				},
			},
			{
				ID:   4,
				Name: "gpt-no-path",
				Extra: models.AIModelExtra{
					DisableSync:       false,
					SyncModelInfoPath: "",
				},
			},
		},
	}
	task := NewPriceSyncTask(
		&log.Logger{Logger: zap.NewNop()},
		newTestPriceStore(),
		repo,
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got := len(repo.updatedModels); got != 2 {
		t.Fatalf("expected 2 updated models, got %d", got)
	}

	model1 := mustFindModelByID(t, repo.models, 1)
	if model1.Extra.SyncModelInfoPath != "openai.models.gpt-4o" {
		t.Fatalf("expected trimmed path, got %q", model1.Extra.SyncModelInfoPath)
	}
	assertFloatNear(t, model1.Pricing.InputPrice, 5)
	assertFloatNear(t, model1.Pricing.OutputPrice, 15)
	assertFloatNear(t, model1.Pricing.CacheReadPrice, 0.5)
	assertFloatNear(t, model1.Pricing.CacheCreatePrice, 1)
	assertFloatNear(t, model1.Pricing.Over200KInputPrice, 6)
	assertFloatNear(t, model1.Pricing.Over200KOutputPrice, 18)
	assertFloatNear(t, model1.Pricing.Over200KCacheReadPrice, 0.6)
	assertFloatNear(t, model1.Pricing.Over200KCacheCreatePrice, 1.2)

	model2 := mustFindModelByID(t, repo.models, 2)
	if model2.Extra.SyncModelInfoPath != "openai.models.gpt-4.1-mini" {
		t.Fatalf("expected discovered path, got %q", model2.Extra.SyncModelInfoPath)
	}
	assertFloatNear(t, model2.Pricing.InputPrice, 0.4)
	assertFloatNear(t, model2.Pricing.OutputPrice, 1.2)
	assertFloatNear(t, model2.Pricing.CacheReadPrice, 0.04)
	assertFloatNear(t, model2.Pricing.CacheCreatePrice, 0.08)
	assertFloatNear(t, model2.Pricing.Over200KInputPrice, 0.8)
	assertFloatNear(t, model2.Pricing.Over200KOutputPrice, 2.4)
	assertFloatNear(t, model2.Pricing.Over200KCacheReadPrice, 0.08)
	assertFloatNear(t, model2.Pricing.Over200KCacheCreatePrice, 0.16)

	model3 := mustFindModelByID(t, repo.models, 3)
	assertFloatNear(t, model3.Pricing.InputPrice, 0)

	statsAny, err := task.CurrentStats()
	if err != nil {
		t.Fatalf("CurrentStats failed: %v", err)
	}
	stats, ok := statsAny.(PriceSyncTaskStats)
	if !ok {
		t.Fatalf("unexpected stats type: %T", statsAny)
	}
	if stats.TotalModels != 4 || stats.SyncedModels != 2 || stats.UpdatedModels != 2 || stats.SkippedModels != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.LastRunAt == nil || stats.UpdatedAt == nil {
		t.Fatalf("expected non-nil run timestamps: %+v", stats)
	}
	if stats.LastError != "" {
		t.Fatalf("expected empty LastError, got %q", stats.LastError)
	}
}

func TestPriceSyncTask_Run_ReturnsUpdateError(t *testing.T) {
	wantErr := errors.New("update failed")
	repo := &testPriceSyncModelRepo{
		models: []*models.AIModel{
			{
				ID:   1,
				Name: "gpt-4o",
				Extra: models.AIModelExtra{
					DisableSync:       false,
					SyncModelInfoPath: "openai.models.gpt-4o",
				},
			},
		},
		updateErr: wantErr,
	}
	task := NewPriceSyncTask(
		&log.Logger{Logger: zap.NewNop()},
		newTestPriceStore(),
		repo,
	)

	gotErr := task.Run(context.Background())
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, gotErr)
	}

	statsAny, err := task.CurrentStats()
	if err != nil {
		t.Fatalf("CurrentStats failed: %v", err)
	}
	stats := statsAny.(PriceSyncTaskStats)
	if stats.LastError == "" {
		t.Fatalf("expected LastError to be recorded, got empty")
	}
	if stats.UpdatedModels != 0 {
		t.Fatalf("expected no successful update, got %d", stats.UpdatedModels)
	}
}

func TestPriceSyncTask_Run_SkipsWhenPathMissing(t *testing.T) {
	repo := &testPriceSyncModelRepo{
		models: []*models.AIModel{
			{
				ID:   1,
				Name: "gpt-x",
				Extra: models.AIModelExtra{
					DisableSync:       false,
					SyncModelInfoPath: "openai.models.not-found",
				},
			},
		},
	}
	task := NewPriceSyncTask(
		&log.Logger{Logger: zap.NewNop()},
		newTestPriceStore(),
		repo,
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(repo.updatedModels) != 0 {
		t.Fatalf("expected no updates, got %d", len(repo.updatedModels))
	}
}

type testPriceSyncModelRepo struct {
	models        []*models.AIModel
	updatedModels []*models.AIModel
	listErr       error
	updateErr     error
}

func (r *testPriceSyncModelRepo) Create(ctx context.Context, model *models.AIModel) error {
	panic("unexpected call to Create")
}

func (r *testPriceSyncModelRepo) BatchCreate(ctx context.Context, modelsList []*models.AIModel) error {
	panic("unexpected call to BatchCreate")
}

func (r *testPriceSyncModelRepo) GetByID(ctx context.Context, id int64) (*models.AIModel, error) {
	panic("unexpected call to GetByID")
}

func (r *testPriceSyncModelRepo) GetByName(ctx context.Context, name string) (*models.AIModel, error) {
	panic("unexpected call to GetByName")
}

func (r *testPriceSyncModelRepo) GetExistingNames(ctx context.Context, names []string) ([]string, error) {
	panic("unexpected call to GetExistingNames")
}

func (r *testPriceSyncModelRepo) List(ctx context.Context, opt repository.AIModelListOption) ([]*models.AIModel, int64, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	page := opt.Page
	if page <= 0 {
		page = 1
	}
	pageSize := opt.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(r.models) {
		return []*models.AIModel{}, int64(len(r.models)), nil
	}
	end := start + pageSize
	if end > len(r.models) {
		end = len(r.models)
	}
	return r.models[start:end], int64(len(r.models)), nil
}

func (r *testPriceSyncModelRepo) Update(ctx context.Context, model *models.AIModel) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if model == nil {
		return nil
	}
	r.updatedModels = append(r.updatedModels, cloneModel(model))
	for i := range r.models {
		if r.models[i] != nil && r.models[i].ID == model.ID {
			r.models[i] = cloneModel(model)
			break
		}
	}
	return nil
}

func (r *testPriceSyncModelRepo) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	panic("unexpected call to SetEnabled")
}

func (r *testPriceSyncModelRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected call to Delete")
}

func (r *testPriceSyncModelRepo) Exists(ctx context.Context, id int64) (bool, error) {
	panic("unexpected call to Exists")
}

func newTestPriceStore() *store.ModelPriceStore {
	raw := []byte(`{
		"openai": {
			"models": {
				"gpt-4o": {
					"cost": {
						"input": 5,
						"output": 15,
						"cache_read": 0.5,
						"cache_write": 1,
						"context_over_200k": {
							"input": 6,
							"output": 18,
							"cache_read": 0.6,
							"cache_write": 1.2
						}
					}
				},
				"gpt-4.1-mini": {
					"cost": {
						"input": 0.4,
						"output": 1.2,
						"cache_read": 0.04,
						"cache_write": 0.08,
						"context_over_200k": {
							"input": 0.8,
							"output": 2.4,
							"cache_read": 0.08,
							"cache_write": 0.16
						}
					}
				}
			}
		}
	}`)
	return &store.ModelPriceStore{
		RawData:         raw,
		LatestFetchTime: time.Now(),
		Models: []string{
			"openai#gpt-4o",
			"openai#gpt-4.1-mini",
		},
	}
}

func mustFindModelByID(t *testing.T, items []*models.AIModel, id int64) *models.AIModel {
	t.Helper()
	for i := range items {
		if items[i] != nil && items[i].ID == id {
			return items[i]
		}
	}
	t.Fatalf("model id %d not found", id)
	return nil
}

func assertFloatNear(t *testing.T, got, want float64) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > 1e-9 {
		t.Fatalf("unexpected float: got %v want %v", got, want)
	}
}

func cloneModel(in *models.AIModel) *models.AIModel {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}
