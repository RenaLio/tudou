package service

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type GroupRegistryReloader interface {
	ReloadGroup(group *models.ChannelGroup)
	UnregisterGroup(groupID int64)
}

type ChannelGroupService interface {
	Create(ctx context.Context, req v1.CreateChannelGroupRequest) (*v1.ChannelGroupResponse, error)
	BatchCreate(ctx context.Context, reqs []v1.CreateChannelGroupRequest) ([]v1.ChannelGroupResponse, error)
	GetByID(ctx context.Context, id int64) (*v1.ChannelGroupResponse, error)
	GetByName(ctx context.Context, name string) (*v1.ChannelGroupResponse, error)
	List(ctx context.Context, req v1.ListChannelGroupsRequest) (*v1.ListResponse[v1.ChannelGroupResponse], error)
	Update(ctx context.Context, id int64, req v1.UpdateChannelGroupRequest) (*v1.ChannelGroupResponse, error)
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

type channelGroupService struct {
	*Service
	repo     repository.ChannelGroupRepo
	registry GroupRegistryReloader
}

func NewChannelGroupService(base *Service, repo repository.ChannelGroupRepo, registry GroupRegistryReloader) ChannelGroupService {
	return &channelGroupService{
		Service:  base,
		repo:     repo,
		registry: registry,
	}
}

func (s *channelGroupService) Create(ctx context.Context, req v1.CreateChannelGroupRequest) (*v1.ChannelGroupResponse, error) {
	group, err := s.buildGroupByCreateReq(req)
	if err != nil {
		return nil, err
	}

	if err = s.repo.Create(ctx, group); err != nil {
		return nil, err
	}
	if s.registry != nil {
		s.registry.ReloadGroup(group)
	}
	resp := toChannelGroupResponse(group)
	return &resp, nil
}

func (s *channelGroupService) BatchCreate(ctx context.Context, reqs []v1.CreateChannelGroupRequest) ([]v1.ChannelGroupResponse, error) {
	if len(reqs) == 0 {
		return []v1.ChannelGroupResponse{}, nil
	}
	groups := make([]*models.ChannelGroup, 0, len(reqs))
	for _, req := range reqs {
		group, err := s.buildGroupByCreateReq(req)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if err := s.repo.BatchCreate(ctx, groups); err != nil {
		return nil, err
	}
	resp := make([]v1.ChannelGroupResponse, 0, len(groups))
	for _, group := range groups {
		resp = append(resp, toChannelGroupResponse(group))
	}
	return resp, nil
}

func (s *channelGroupService) GetByID(ctx context.Context, id int64) (*v1.ChannelGroupResponse, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toChannelGroupResponse(group)
	return &resp, nil
}

func (s *channelGroupService) GetByName(ctx context.Context, name string) (*v1.ChannelGroupResponse, error) {
	group, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	resp := toChannelGroupResponse(group)
	return &resp, nil
}

func (s *channelGroupService) List(ctx context.Context, req v1.ListChannelGroupsRequest) (*v1.ListResponse[v1.ChannelGroupResponse], error) {
	opt := repository.ChannelGroupListOption{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		Keyword:  req.Keyword,
	}
	items, total, err := s.repo.List(ctx, opt)
	if err != nil {
		return nil, err
	}

	respItems := make([]v1.ChannelGroupResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toChannelGroupResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.ChannelGroupResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func (s *channelGroupService) Update(ctx context.Context, id int64, req v1.UpdateChannelGroupRequest) (*v1.ChannelGroupResponse, error) {
	group, err := s.repo.GetByIDWithChannels(ctx, id)
	if err != nil {
		return nil, err
	}
	patchGroupByUpdateReq(group, req)
	if strings.TrimSpace(group.Name) == "" {
		return nil, errors.New("name is required")
	}

	if err = s.repo.Update(ctx, group); err != nil {
		return nil, err
	}
	if s.registry != nil {
		s.registry.ReloadGroup(group)
	}
	resp := toChannelGroupResponse(group)
	return &resp, nil
}

func (s *channelGroupService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.registry != nil {
		s.registry.UnregisterGroup(id)
	}
	return nil
}

func (s *channelGroupService) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}

func (s *channelGroupService) buildGroupByCreateReq(req v1.CreateChannelGroupRequest) (*models.ChannelGroup, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	id := s.NextID()
	if id <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}
	group := &models.ChannelGroup{
		ID:         id,
		Name:       name,
		NameRemark: strings.TrimSpace(req.NameRemark),
	}
	if req.LoadBalanceStrategy != nil {
		group.LoadBalanceStrategy = *req.LoadBalanceStrategy
	} else {
		group.LoadBalanceStrategy = models.LoadBalanceStrategyPerformance
	}
	return group, nil
}

func patchGroupByUpdateReq(group *models.ChannelGroup, req v1.UpdateChannelGroupRequest) {
	if group == nil {
		return
	}
	if req.Name != nil {
		group.Name = strings.TrimSpace(*req.Name)
	}
	if req.NameRemark != nil {
		group.NameRemark = strings.TrimSpace(*req.NameRemark)
	}
	if req.LoadBalanceStrategy != nil {
		group.LoadBalanceStrategy = *req.LoadBalanceStrategy
	}
}

func toChannelGroupResponse(group *models.ChannelGroup) v1.ChannelGroupResponse {
	if group == nil {
		return v1.ChannelGroupResponse{}
	}
	return v1.ChannelGroupResponse{
		ID:                  group.ID,
		Name:                group.Name,
		NameRemark:          group.NameRemark,
		LoadBalanceStrategy: group.LoadBalanceStrategy,
		CreatedAt:           group.CreatedAt,
		UpdatedAt:           group.UpdatedAt,
	}
}
