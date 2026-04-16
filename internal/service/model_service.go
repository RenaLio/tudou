package service

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type AIModelService interface {
	Create(ctx context.Context, req v1.CreateAIModelRequest) (*v1.AIModelResponse, error)
	BatchCreate(ctx context.Context, reqs []v1.CreateAIModelRequest) ([]v1.AIModelResponse, error)
	GetByID(ctx context.Context, id int64) (*v1.AIModelResponse, error)
	GetByName(ctx context.Context, name string) (*v1.AIModelResponse, error)
	List(ctx context.Context, req v1.ListAIModelsRequest) (*v1.ListResponse[v1.AIModelResponse], error)
	Update(ctx context.Context, id int64, req v1.UpdateAIModelRequest) (*v1.AIModelResponse, error)
	SetEnabled(ctx context.Context, id int64, req v1.SetAIModelEnabledRequest) (*v1.AIModelResponse, error)
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

type aiModelService struct {
	*Service
	repo repository.AIModelRepo
}

func NewAIModelService(base *Service, repo repository.AIModelRepo) AIModelService {
	return &aiModelService{
		Service: base,
		repo:    repo,
	}
}

func (s *aiModelService) Create(ctx context.Context, req v1.CreateAIModelRequest) (*v1.AIModelResponse, error) {
	model, err := s.buildModelByCreateReq(req)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	resp := toAIModelResponse(model)
	return &resp, nil
}

func (s *aiModelService) BatchCreate(ctx context.Context, reqs []v1.CreateAIModelRequest) ([]v1.AIModelResponse, error) {
	if len(reqs) == 0 {
		return []v1.AIModelResponse{}, nil
	}
	modelsList := make([]*models.AIModel, 0, len(reqs))
	for _, req := range reqs {
		model, err := s.buildModelByCreateReq(req)
		if err != nil {
			return nil, err
		}
		modelsList = append(modelsList, model)
	}

	if err := s.repo.BatchCreate(ctx, modelsList); err != nil {
		return nil, err
	}
	respItems := make([]v1.AIModelResponse, 0, len(modelsList))
	for i := range modelsList {
		respItems = append(respItems, toAIModelResponse(modelsList[i]))
	}
	return respItems, nil
}

func (s *aiModelService) GetByID(ctx context.Context, id int64) (*v1.AIModelResponse, error) {
	model, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toAIModelResponse(model)
	return &resp, nil
}

func (s *aiModelService) GetByName(ctx context.Context, name string) (*v1.AIModelResponse, error) {
	model, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	resp := toAIModelResponse(model)
	return &resp, nil
}

func (s *aiModelService) List(ctx context.Context, req v1.ListAIModelsRequest) (*v1.ListResponse[v1.AIModelResponse], error) {
	opt := repository.AIModelListOption{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		Keyword:  req.Keyword,
	}

	items, total, err := s.repo.List(ctx, opt)
	if err != nil {
		return nil, err
	}

	respItems := make([]v1.AIModelResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toAIModelResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.AIModelResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func (s *aiModelService) Update(ctx context.Context, id int64, req v1.UpdateAIModelRequest) (*v1.AIModelResponse, error) {
	model, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	patchModelByUpdateReq(model, req)
	if strings.TrimSpace(model.Name) == "" {
		return nil, errors.New("name is required")
	}
	if err = s.repo.Update(ctx, model); err != nil {
		return nil, err
	}
	resp := toAIModelResponse(model)
	return &resp, nil
}

func (s *aiModelService) SetEnabled(ctx context.Context, id int64, req v1.SetAIModelEnabledRequest) (*v1.AIModelResponse, error) {
	if err := s.repo.SetEnabled(ctx, id, req.Enabled); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *aiModelService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *aiModelService) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}

func (s *aiModelService) buildModelByCreateReq(req v1.CreateAIModelRequest) (*models.AIModel, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	id := s.NextID()
	if id <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}
	model := &models.AIModel{
		ID:          id,
		Name:        name,
		Type:        models.ModelTypeChat,
		Description: strings.TrimSpace(req.Description),
		Pricing:     req.Pricing,
		PricingType: req.PricingType,
		IsEnabled:   true,
	}
	if model.PricingType == "" {
		model.PricingType = models.ModelPricingTypeTokens
	}
	return model, nil
}

func patchModelByUpdateReq(model *models.AIModel, req v1.UpdateAIModelRequest) {
	if model == nil {
		return
	}
	if req.Name != nil {
		model.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		model.Description = strings.TrimSpace(*req.Description)
	}
	if req.Pricing != nil {
		model.Pricing = *req.Pricing
	}
	if req.PricingType != nil {
		model.PricingType = *req.PricingType
	}
}

func toAIModelResponse(model *models.AIModel) v1.AIModelResponse {
	if model == nil {
		return v1.AIModelResponse{}
	}
	return v1.AIModelResponse{
		ID:           model.ID,
		Name:         model.Name,
		Type:         model.Type,
		Description:  model.Description,
		Pricing:      model.Pricing,
		Capabilities: model.Capabilities,
		PricingType:  model.PricingType,
		IsEnabled:    model.IsEnabled,
		Extra:        model.Extra,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
