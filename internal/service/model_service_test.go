package service

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/sid"
	"github.com/RenaLio/tudou/internal/repository"
)

func TestBuildModelByCreateReq_PopulatesExtra(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.Sid.Id = 1
	svc := &AIModelService{
		Service: &Service{
			sid: sid.NewSid(cfg),
		},
	}

	req := v1.CreateAIModelRequest{
		Name:        "  gpt-4.1-mini  ",
		Description: "  fast model  ",
		PricingType: models.ModelPricingTypeTokens,
		Extra: models.AIModelExtra{
			SyncModelInfoPath: "openai.models.gpt-4.1-mini",
			DisableSync:       false,
		},
	}

	model, err := svc.buildModelByCreateReq(req)
	if err != nil {
		t.Fatalf("buildModelByCreateReq failed: %v", err)
	}
	if model.Name != "gpt-4.1-mini" {
		t.Fatalf("unexpected name: %q", model.Name)
	}
	if model.Description != "fast model" {
		t.Fatalf("unexpected description: %q", model.Description)
	}
	if model.Extra != req.Extra {
		t.Fatalf("unexpected extra: %+v", model.Extra)
	}
	if model.Pricing.LongContextTokens != 256_000 {
		t.Fatalf("unexpected long context tokens: got=%d want=256000", model.Pricing.LongContextTokens)
	}
}

func TestPatchModelByUpdateReq_PopulatesExtra(t *testing.T) {
	model := &models.AIModel{
		Name: "gpt-4o",
		Extra: models.AIModelExtra{
			SyncModelInfoPath: "openai.models.gpt-4o",
			DisableSync:       false,
		},
	}

	nextExtra := &models.AIModelExtra{
		SyncModelInfoPath: "openai.models.gpt-5",
		DisableSync:       true,
	}
	req := v1.UpdateAIModelRequest{
		Extra: nextExtra,
	}

	patchModelByUpdateReq(model, req)
	if model.Extra != *nextExtra {
		t.Fatalf("unexpected extra after patch: %+v", model.Extra)
	}
}

func TestBuildModelByCreateReq_DefaultsLongContextTokens(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.Sid.Id = 1
	svc := &AIModelService{
		Service: &Service{
			sid: sid.NewSid(cfg),
		},
	}

	model, err := svc.buildModelByCreateReq(v1.CreateAIModelRequest{
		Name: "gpt-4o",
	})
	if err != nil {
		t.Fatalf("buildModelByCreateReq failed: %v", err)
	}
	if model.Pricing.LongContextTokens != 256_000 {
		t.Fatalf("unexpected long context tokens: got=%d want=256000", model.Pricing.LongContextTokens)
	}
}

func TestPatchModelByUpdateReq_DefaultsLongContextTokens(t *testing.T) {
	model := &models.AIModel{
		Name: "gpt-4o",
		Pricing: models.ModelPricing{
			LongContextTokens: 256_000,
		},
	}

	nextPricing := &models.ModelPricing{}
	req := v1.UpdateAIModelRequest{
		Pricing: nextPricing,
	}

	patchModelByUpdateReq(model, req)
	if model.Pricing.LongContextTokens != 256_000 {
		t.Fatalf("unexpected long context tokens after patch: got=%d want=256000", model.Pricing.LongContextTokens)
	}
}

type testModelRepoForNameFlow struct {
	getByNameFn    func(ctx context.Context, name string) (*models.AIModel, error)
	updateFn       func(ctx context.Context, model *models.AIModel) error
	existsByNameFn func(ctx context.Context, name string) (bool, error)
	deleteByNameFn func(ctx context.Context, name string) error
}

var _ repository.AIModelRepo = (*testModelRepoForNameFlow)(nil)

func (r *testModelRepoForNameFlow) Create(context.Context, *models.AIModel) error {
	panic("not implemented")
}

func (r *testModelRepoForNameFlow) BatchCreate(context.Context, []*models.AIModel) error {
	panic("not implemented")
}

func (r *testModelRepoForNameFlow) GetByName(ctx context.Context, name string) (*models.AIModel, error) {
	if r.getByNameFn == nil {
		panic("not implemented")
	}
	return r.getByNameFn(ctx, name)
}

func (r *testModelRepoForNameFlow) GetExistingNames(context.Context, []string) ([]string, error) {
	panic("not implemented")
}

func (r *testModelRepoForNameFlow) List(context.Context, repository.AIModelListOption) ([]*models.AIModel, int64, error) {
	panic("not implemented")
}

func (r *testModelRepoForNameFlow) Update(ctx context.Context, model *models.AIModel) error {
	if r.updateFn == nil {
		return nil
	}
	return r.updateFn(ctx, model)
}

func (r *testModelRepoForNameFlow) DeleteByName(ctx context.Context, name string) error {
	if r.deleteByNameFn == nil {
		panic("not implemented")
	}
	return r.deleteByNameFn(ctx, name)
}

func (r *testModelRepoForNameFlow) DeleteByNames(context.Context, []string) (int64, error) {
	panic("not implemented")
}

func (r *testModelRepoForNameFlow) ExistsByName(ctx context.Context, name string) (bool, error) {
	if r.existsByNameFn == nil {
		panic("not implemented")
	}
	return r.existsByNameFn(ctx, name)
}

func TestAIModelService_Update_NameImmutable(t *testing.T) {
	model := &models.AIModel{
		ID:   1,
		Name: "gpt-4o",
	}
	repo := &testModelRepoForNameFlow{
		getByNameFn: func(ctx context.Context, name string) (*models.AIModel, error) {
			return model, nil
		},
	}
	svc := &AIModelService{repo: repo}

	nextName := "gpt-4.1"
	_, err := svc.Update(context.Background(), "gpt-4o", v1.UpdateAIModelRequest{
		Name: &nextName,
	})
	if err == nil || err.Error() != "model name is immutable" {
		t.Fatalf("expected immutable-name error, got: %v", err)
	}
}

func TestAIModelService_Exists_NotFound(t *testing.T) {
	repo := &testModelRepoForNameFlow{
		existsByNameFn: func(ctx context.Context, name string) (bool, error) {
			return false, nil
		},
	}
	svc := &AIModelService{repo: repo}

	ok, err := svc.Exists(context.Background(), "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected exists=false when record not found")
	}
}

func TestAIModelService_Exists_UnexpectedError(t *testing.T) {
	wantErr := errors.New("db timeout")
	repo := &testModelRepoForNameFlow{
		existsByNameFn: func(ctx context.Context, name string) (bool, error) {
			return false, wantErr
		},
	}
	svc := &AIModelService{repo: repo}

	ok, err := svc.Exists(context.Background(), "gpt-4o")
	if ok {
		t.Fatal("expected exists=false on error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: %v", err)
	}
}
