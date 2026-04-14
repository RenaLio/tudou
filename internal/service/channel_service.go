package service

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type ChannelService interface {
	Create(ctx context.Context, req v1.CreateChannelRequest) (*v1.ChannelResponse, error)
	BatchCreate(ctx context.Context, reqs []v1.CreateChannelRequest) ([]v1.ChannelResponse, error)
	GetByID(ctx context.Context, id int64, withGroups bool) (*v1.ChannelResponse, error)
	GetByName(ctx context.Context, name string) (*v1.ChannelResponse, error)
	GetByIDs(ctx context.Context, ids []int64) ([]v1.ChannelResponse, error)
	List(ctx context.Context, req v1.ListChannelsRequest) (*v1.ListResponse[v1.ChannelResponse], error)
	Update(ctx context.Context, id int64, req v1.UpdateChannelRequest) (*v1.ChannelResponse, error)
	UpdateStatus(ctx context.Context, id int64, req v1.SetChannelStatusRequest) (*v1.ChannelResponse, error)
	Delete(ctx context.Context, id int64) error
	ReplaceGroups(ctx context.Context, channelID int64, req v1.ReplaceChannelGroupsRequest) (*v1.ChannelResponse, error)
	Exists(ctx context.Context, id int64) (bool, error)
}

type channelService struct {
	*Service
	repo repository.ChannelRepo
}

func NewChannelService(base *Service, repo repository.ChannelRepo) ChannelService {
	return &channelService{
		Service: base,
		repo:    repo,
	}
}

func (s *channelService) Create(ctx context.Context, req v1.CreateChannelRequest) (*v1.ChannelResponse, error) {
	channel, err := s.buildChannelByCreateReq(req)
	if err != nil {
		return nil, err
	}

	if len(req.GroupIDs) == 0 {
		if err = s.repo.Create(ctx, channel); err != nil {
			return nil, err
		}
		resp := toChannelResponse(channel)
		return &resp, nil
	}

	if err = s.Transaction(ctx, func(txCtx context.Context) error {
		if txErr := s.repo.Create(txCtx, channel); txErr != nil {
			return txErr
		}
		return s.repo.ReplaceGroups(txCtx, channel.ID, req.GroupIDs)
	}); err != nil {
		return nil, err
	}

	latest, err := s.repo.GetByIDWithGroups(ctx, channel.ID)
	if err != nil {
		return nil, err
	}
	resp := toChannelResponse(latest)
	return &resp, nil
}

func (s *channelService) BatchCreate(ctx context.Context, reqs []v1.CreateChannelRequest) ([]v1.ChannelResponse, error) {
	if len(reqs) == 0 {
		return []v1.ChannelResponse{}, nil
	}
	channels := make([]*models.Channel, 0, len(reqs))
	for _, req := range reqs {
		channel, err := s.buildChannelByCreateReq(req)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	if err := s.repo.BatchCreate(ctx, channels); err != nil {
		return nil, err
	}
	resp := make([]v1.ChannelResponse, 0, len(channels))
	for i := range channels {
		resp = append(resp, toChannelResponse(channels[i]))
	}
	return resp, nil
}

func (s *channelService) GetByID(ctx context.Context, id int64, withGroups bool) (*v1.ChannelResponse, error) {
	var (
		channel *models.Channel
		err     error
	)
	if withGroups {
		channel, err = s.repo.GetByIDWithGroups(ctx, id)
	} else {
		channel, err = s.repo.GetByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	resp := toChannelResponse(channel)
	return &resp, nil
}

func (s *channelService) GetByName(ctx context.Context, name string) (*v1.ChannelResponse, error) {
	channel, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	resp := toChannelResponse(channel)
	return &resp, nil
}

func (s *channelService) GetByIDs(ctx context.Context, ids []int64) ([]v1.ChannelResponse, error) {
	items, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	resp := make([]v1.ChannelResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, toChannelResponse(item))
	}
	return resp, nil
}

func (s *channelService) List(ctx context.Context, req v1.ListChannelsRequest) (*v1.ListResponse[v1.ChannelResponse], error) {
	opt := repository.ChannelListOption{
		Page:          req.Page,
		PageSize:      req.PageSize,
		OrderBy:       req.OrderBy,
		Keyword:       req.Keyword,
		GroupID:       req.GroupID,
		Status:        req.Status,
		OnlyAvailable: req.OnlyAvailable,
		PreloadGroups: req.PreloadGroups,
	}
	if req.Type != "" {
		opt.Type = models.ChannelType(req.Type)
	}

	items, total, err := s.repo.List(ctx, opt)
	if err != nil {
		return nil, err
	}
	respItems := make([]v1.ChannelResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toChannelResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.ChannelResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func (s *channelService) Update(ctx context.Context, id int64, req v1.UpdateChannelRequest) (*v1.ChannelResponse, error) {
	channel, err := s.repo.GetByIDWithGroups(ctx, id)
	if err != nil {
		return nil, err
	}
	patchChannelByUpdateReq(channel, req)
	if strings.TrimSpace(channel.Name) == "" || strings.TrimSpace(channel.BaseURL) == "" || strings.TrimSpace(channel.APIKey) == "" {
		return nil, errors.New("name/baseURL/apiKey are required")
	}

	if !req.ReplaceGroups {
		if err = s.repo.Update(ctx, channel); err != nil {
			return nil, err
		}
		resp := toChannelResponse(channel)
		return &resp, nil
	}

	if err = s.Transaction(ctx, func(txCtx context.Context) error {
		if txErr := s.repo.Update(txCtx, channel); txErr != nil {
			return txErr
		}
		return s.repo.ReplaceGroups(txCtx, channel.ID, req.GroupIDs)
	}); err != nil {
		return nil, err
	}

	latest, err := s.repo.GetByIDWithGroups(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toChannelResponse(latest)
	return &resp, nil
}

func (s *channelService) UpdateStatus(ctx context.Context, id int64, req v1.SetChannelStatusRequest) (*v1.ChannelResponse, error) {
	if err := s.repo.UpdateStatus(ctx, id, req.Status); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id, true)
}

func (s *channelService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *channelService) ReplaceGroups(ctx context.Context, channelID int64, req v1.ReplaceChannelGroupsRequest) (*v1.ChannelResponse, error) {
	if err := s.repo.ReplaceGroups(ctx, channelID, req.GroupIDs); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, channelID, true)
}

func (s *channelService) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}

func (s *channelService) buildChannelByCreateReq(req v1.CreateChannelRequest) (*models.Channel, error) {
	name := strings.TrimSpace(req.Name)
	baseURL := strings.TrimSpace(req.BaseURL)
	apiKey := strings.TrimSpace(req.APIKey)
	if name == "" || baseURL == "" || apiKey == "" {
		return nil, errors.New("name/baseURL/apiKey are required")
	}
	id := s.NextID()
	if id <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}

	channel := &models.Channel{
		ID:          id,
		Type:        req.Type,
		Name:        name,
		BaseURL:     baseURL,
		APIKey:      apiKey,
		Remark:      strings.TrimSpace(req.Remark),
		Tag:         strings.TrimSpace(req.Tag),
		Model:       strings.TrimSpace(req.Model),
		CustomModel: strings.TrimSpace(req.CustomModel),
		Settings:    req.Settings,
		Extra:       req.Extra,
		ExpiredAt:   req.ExpiredAt,
	}
	if req.Weight == nil {
		channel.Weight = 100
	} else {
		channel.Weight = *req.Weight
	}
	if req.Status == nil {
		channel.Status = 1
	} else {
		channel.Status = *req.Status
	}
	if req.PriceRate == nil {
		channel.PriceRate = 1
	} else {
		channel.PriceRate = *req.PriceRate
	}
	return channel, nil
}

func patchChannelByUpdateReq(channel *models.Channel, req v1.UpdateChannelRequest) {
	if channel == nil {
		return
	}
	if req.Type != nil {
		channel.Type = *req.Type
	}
	if req.Name != nil {
		channel.Name = strings.TrimSpace(*req.Name)
	}
	if req.BaseURL != nil {
		channel.BaseURL = strings.TrimSpace(*req.BaseURL)
	}
	if req.APIKey != nil {
		channel.APIKey = strings.TrimSpace(*req.APIKey)
	}
	if req.Weight != nil {
		channel.Weight = *req.Weight
	}
	if req.Status != nil {
		channel.Status = *req.Status
	}
	if req.Remark != nil {
		channel.Remark = strings.TrimSpace(*req.Remark)
	}
	if req.Tag != nil {
		channel.Tag = strings.TrimSpace(*req.Tag)
	}
	if req.Model != nil {
		channel.Model = strings.TrimSpace(*req.Model)
	}
	if req.CustomModel != nil {
		channel.CustomModel = strings.TrimSpace(*req.CustomModel)
	}
	if req.Settings != nil {
		channel.Settings = *req.Settings
	}
	if req.Extra != nil {
		channel.Extra = *req.Extra
	}
	if req.PriceRate != nil {
		channel.PriceRate = *req.PriceRate
	}
	if req.ExpiredAt != nil {
		channel.ExpiredAt = *req.ExpiredAt
	}
}

func toChannelResponse(channel *models.Channel) v1.ChannelResponse {
	if channel == nil {
		return v1.ChannelResponse{}
	}
	groupIDs := make([]int64, 0, len(channel.Groups))
	for _, group := range channel.Groups {
		groupIDs = append(groupIDs, group.ID)
	}
	return v1.ChannelResponse{
		ID:          channel.ID,
		Type:        channel.Type,
		Name:        channel.Name,
		BaseURL:     channel.BaseURL,
		Weight:      channel.Weight,
		Status:      channel.Status,
		Remark:      channel.Remark,
		Tag:         channel.Tag,
		Model:       channel.Model,
		CustomModel: channel.CustomModel,
		Settings:    channel.Settings,
		Extra:       channel.Extra,
		PriceRate:   channel.PriceRate,
		ExpiredAt:   channel.ExpiredAt,
		CreatedAt:   channel.CreatedAt,
		UpdatedAt:   channel.UpdatedAt,
		GroupIDs:    groupIDs,
	}
}
