package service

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type SystemConfigService interface {
	Create(ctx context.Context, req v1.CreateSystemConfigRequest) (*v1.SystemConfigResponse, error)
	Upsert(ctx context.Context, req v1.UpsertSystemConfigRequest) (*v1.SystemConfigResponse, error)
	GetByID(ctx context.Context, id int64) (*v1.SystemConfigResponse, error)
	GetByKey(ctx context.Context, key string) (*v1.SystemConfigResponse, error)
	GetString(ctx context.Context, key string, def string) (string, error)
	List(ctx context.Context, req v1.ListSystemConfigsRequest) (*v1.ListResponse[v1.SystemConfigResponse], error)
	SetValueByKey(ctx context.Context, key string, req v1.SetSystemConfigValueRequest) (*v1.SystemConfigResponse, error)
	DeleteByKey(ctx context.Context, key string) error
	InitDefaults(ctx context.Context) error
}

type systemConfigService struct {
	*Service
	repo repository.SystemConfigRepo
}

func NewSystemConfigService(base *Service, repo repository.SystemConfigRepo) SystemConfigService {
	return &systemConfigService{
		Service: base,
		repo:    repo,
	}
}

func (s *systemConfigService) Create(ctx context.Context, req v1.CreateSystemConfigRequest) (*v1.SystemConfigResponse, error) {
	config, err := s.buildConfigByCreateReq(req)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, config); err != nil {
		return nil, err
	}
	resp := toSystemConfigResponse(config)
	return &resp, nil
}

func (s *systemConfigService) Upsert(ctx context.Context, req v1.UpsertSystemConfigRequest) (*v1.SystemConfigResponse, error) {
	config, err := s.buildConfigByUpsertReq(req)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Upsert(ctx, config); err != nil {
		return nil, err
	}
	return s.GetByKey(ctx, config.Key)
}

func (s *systemConfigService) GetByID(ctx context.Context, id int64) (*v1.SystemConfigResponse, error) {
	config, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toSystemConfigResponse(config)
	return &resp, nil
}

func (s *systemConfigService) GetByKey(ctx context.Context, key string) (*v1.SystemConfigResponse, error) {
	config, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	resp := toSystemConfigResponse(config)
	return &resp, nil
}

func (s *systemConfigService) List(ctx context.Context, req v1.ListSystemConfigsRequest) (*v1.ListResponse[v1.SystemConfigResponse], error) {
	opt := repository.SystemConfigListOption{
		Page:        req.Page,
		PageSize:    req.PageSize,
		OrderBy:     req.OrderBy,
		Keyword:     req.Keyword,
		OnlyVisible: req.OnlyVisible,
	}
	if req.Scope != "" {
		scope := models.ConfigScope(req.Scope)
		opt.Scope = &scope
	}

	items, total, err := s.repo.List(ctx, opt)
	if err != nil {
		return nil, err
	}

	respItems := make([]v1.SystemConfigResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toSystemConfigResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.SystemConfigResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func (s *systemConfigService) SetValueByKey(ctx context.Context, key string, req v1.SetSystemConfigValueRequest) (*v1.SystemConfigResponse, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("key is required")
	}
	if err := s.repo.SetValueByKey(ctx, key, req.Value); err != nil {
		return nil, err
	}
	return s.GetByKey(ctx, key)
}

func (s *systemConfigService) DeleteByKey(ctx context.Context, key string) error {
	return s.repo.DeleteByKey(ctx, key)
}

func (s *systemConfigService) InitDefaults(ctx context.Context) error {
	return nil
	//return s.repo.InitDefaults(ctx)
}

func (s *systemConfigService) GetString(ctx context.Context, key string, def string) (string, error) {
	config, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return def, err
	}
	val, ok := config.Value.ValueData.(string)
	if !ok {
		return def, errors.New("config value is not a string")
	}
	return val, nil
}

func (s *systemConfigService) buildConfigByCreateReq(req v1.CreateSystemConfigRequest) (*models.SystemConfig, error) {
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return nil, errors.New("key is required")
	}
	id := s.NextID()
	if id <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}
	config := &models.SystemConfig{
		ID:          id,
		Key:         key,
		Value:       models.ConfigValue{ValueData: req.Value},
		Type:        req.Type,
		Scope:       req.Scope,
		Description: strings.TrimSpace(req.Description),
	}
	if config.Type == "" {
		config.Type = models.ConfigTypeString
	}
	if config.Scope == "" {
		config.Scope = models.ConfigScopeSystem
	}
	if req.IsEditable == nil {
		config.IsEditable = true
	} else {
		config.IsEditable = *req.IsEditable
	}
	if req.IsVisible == nil {
		config.IsVisible = true
	} else {
		config.IsVisible = *req.IsVisible
	}
	if req.Sort != nil {
		config.Sort = *req.Sort
	}
	return config, nil
}

func (s *systemConfigService) buildConfigByUpsertReq(req v1.UpsertSystemConfigRequest) (*models.SystemConfig, error) {
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return nil, errors.New("key is required")
	}
	config := &models.SystemConfig{
		ID:          s.NextID(),
		Key:         key,
		Value:       models.ConfigValue{ValueData: req.Value},
		Type:        req.Type,
		Scope:       req.Scope,
		Description: strings.TrimSpace(req.Description),
	}
	if config.Type == "" {
		config.Type = models.ConfigTypeString
	}
	if config.Scope == "" {
		config.Scope = models.ConfigScopeSystem
	}
	if req.IsEditable == nil {
		config.IsEditable = true
	} else {
		config.IsEditable = *req.IsEditable
	}
	if req.IsVisible == nil {
		config.IsVisible = true
	} else {
		config.IsVisible = *req.IsVisible
	}
	if req.Sort != nil {
		config.Sort = *req.Sort
	}
	if config.ID <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}
	return config, nil
}

func toSystemConfigResponse(config *models.SystemConfig) v1.SystemConfigResponse {
	if config == nil {
		return v1.SystemConfigResponse{}
	}
	return v1.SystemConfigResponse{
		ID:          config.ID,
		Key:         config.Key,
		Value:       config.Value.ValueData,
		Type:        config.Type,
		Scope:       config.Scope,
		Description: config.Description,
		IsEditable:  config.IsEditable,
		IsVisible:   config.IsVisible,
		Sort:        config.Sort,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}
}
