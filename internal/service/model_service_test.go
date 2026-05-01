package service

import (
	"testing"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/sid"
)

func TestBuildModelByCreateReq_PopulatesExtra(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.Sid.Id = 1
	svc := &aiModelService{
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
