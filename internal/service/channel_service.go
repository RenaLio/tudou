package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/store"
)

type LBRegistryReloader interface {
	ReloadChannel(channel *models.Channel)
	UnregisterChannel(channelID int64)
	ReloadGroup(group *models.ChannelGroup)
}

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
	repo       repository.ChannelRepo
	modelRepo  repository.AIModelRepo
	groupRepo  repository.ChannelGroupRepo
	registry   LBRegistryReloader
	priceStore *store.ModelPriceStore
}

func NewChannelService(
	base *Service,
	repo repository.ChannelRepo,
	modelRepo repository.AIModelRepo,
	registry LBRegistryReloader,
	groupRepo repository.ChannelGroupRepo,
	priceStore *store.ModelPriceStore,
) ChannelService {
	return &channelService{
		Service:    base,
		repo:       repo,
		modelRepo:  modelRepo,
		groupRepo:  groupRepo,
		registry:   registry,
		priceStore: priceStore,
	}
}

func (s *channelService) Create(ctx context.Context, req v1.CreateChannelRequest) (*v1.ChannelResponse, error) {
	channel, err := s.buildChannelByCreateReq(req)
	if err != nil {
		return nil, err
	}

	if err = s.Transaction(ctx, func(txCtx context.Context) error {
		if txErr := s.repo.Create(txCtx, channel); txErr != nil {
			return txErr
		}
		if len(req.GroupIDs) > 0 {
			if txErr := s.repo.ReplaceGroups(txCtx, channel.ID, req.GroupIDs); txErr != nil {
				return txErr
			}
		}
		return s.ensureModelsExist(txCtx, channel)
	}); err != nil {
		return nil, err
	}

	latest, err := s.repo.GetByID(ctx, channel.ID)
	if err != nil {
		return nil, err
	}
	if s.registry != nil {
		s.registry.ReloadChannel(latest)
	}
	if len(req.GroupIDs) > 0 {
		ReloadGroup(ctx, s.groupRepo, s.registry)
	}
	resp := toChannelResponse(latest)
	return &resp, nil
}

func ReloadGroup(ctx context.Context, groupRepo repository.ChannelGroupRepo, registry LBRegistryReloader) error {
	groups, err := groupRepo.PreLoadRegistryData(ctx)
	if err != nil {
		return err
	}
	for _, g := range groups {
		registry.ReloadGroup(g)
	}
	return nil
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

	if err := s.Transaction(ctx, func(txCtx context.Context) error {
		if txErr := s.repo.BatchCreate(txCtx, channels); txErr != nil {
			return txErr
		}
		for _, channel := range channels {
			if txErr := s.ensureModelsExist(txCtx, channel); txErr != nil {
				return txErr
			}
		}
		return nil
	}); err != nil {
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
		PreloadStats:  req.PreloadStats,
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

	// 统一在事务中执行更新、分组关联和模型同步
	if err = s.Transaction(ctx, func(txCtx context.Context) error {
		if txErr := s.repo.Update(txCtx, channel); txErr != nil {
			return txErr
		}
		if len(req.GroupIDs) > 0 {
			if txErr := s.repo.ReplaceGroups(txCtx, channel.ID, req.GroupIDs); txErr != nil {
				return txErr
			}
		}
		return s.ensureModelsExist(txCtx, channel)
	}); err != nil {
		return nil, err
	}

	latest, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.registry != nil {
		s.registry.ReloadChannel(latest)
	}
	if len(req.GroupIDs) > 0 {
		ReloadGroup(ctx, s.groupRepo, s.registry)
	}

	resp := toChannelResponse(latest)
	return &resp, nil
}

func (s *channelService) UpdateStatus(ctx context.Context, id int64, req v1.SetChannelStatusRequest) (*v1.ChannelResponse, error) {
	if err := s.repo.UpdateStatus(ctx, id, *req.Status); err != nil {
		return nil, err
	}
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.registry != nil {
		s.registry.ReloadChannel(ch)
	}
	resp := toChannelResponse(ch)
	return &resp, nil
}

func (s *channelService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.registry != nil {
		s.registry.UnregisterChannel(id)
	}
	return nil
}

func (s *channelService) ReplaceGroups(ctx context.Context, channelID int64, req v1.ReplaceChannelGroupsRequest) (*v1.ChannelResponse, error) {
	if err := s.repo.ReplaceGroups(ctx, channelID, req.GroupIDs); err != nil {
		return nil, err
	}
	ch, err := s.repo.GetByIDWithGroups(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if s.registry != nil {
		s.registry.ReloadChannel(ch)
	}
	resp := toChannelResponse(ch)
	return &resp, nil
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
		ExpiredAt:   req.ExpiredAt,
		Status:      models.ChannelStatusEnabled, // 默认启用
	}
	if req.Settings != nil {
		channel.Settings = *req.Settings
	}
	if req.Extra != nil {
		channel.Extra = *req.Extra
	}
	if req.Weight == nil {
		channel.Weight = 100
	} else {
		channel.Weight = *req.Weight
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
		channel.ExpiredAt = req.ExpiredAt
	}
}

func toChannelResponse(channel *models.Channel) v1.ChannelResponse {
	if channel == nil {
		return v1.ChannelResponse{}
	}
	groupIDs := make([]string, 0, len(channel.Groups))
	for _, group := range channel.Groups {
		groupIDs = append(groupIDs, strconv.FormatInt(group.ID, 10))
	}
	var stats *models.ChannelStats
	if channel.Stats != nil && channel.Stats.ChannelID != 0 {
		s := *channel.Stats
		stats = &s
	}
	groups := make([]v1.ChannelGroupResponse, 0, len(channel.Groups))
	for _, group := range channel.Groups {
		groups = append(groups, toChannelGroupResponse(&group))
	}
	return v1.ChannelResponse{
		ID:          channel.ID,
		Type:        channel.Type,
		Name:        channel.Name,
		BaseURL:     channel.BaseURL,
		APIKey:      channel.APIKey,
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
		Groups:      groups,
		Stats:       new(toChannelStatsResponse(stats)),
	}
}

// ensureModelsExist 从 channel 支持的模型列表中提取所有模型名，
// 对数据库中不存在的模型批量创建 AIModel 记录。
func (s *channelService) ensureModelsExist(ctx context.Context, channel *models.Channel) error {
	if channel == nil {
		return nil
	}
	modelMap := channel.Models()
	if len(modelMap) == 0 {
		return nil
	}

	// 收集去重的模型名（包括调用名和上游名）
	seen := make(map[string]struct{}, len(modelMap)*2)
	names := make([]string, 0, len(modelMap)*2)
	addModelName := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for callModel, upstreamModel := range modelMap {
		addModelName(callModel)
		addModelName(upstreamModel)
	}
	if len(names) == 0 {
		return nil
	}

	// 查出已存在的模型名
	existingNames, err := s.modelRepo.GetExistingNames(ctx, names)
	if err != nil {
		return err
	}
	existSet := make(map[string]struct{}, len(existingNames))
	for _, name := range existingNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		existSet[name] = struct{}{}
	}

	// 筛选出需要创建的
	toCreate := make([]*models.AIModel, 0, len(names)-len(existingNames))
	for _, name := range names {
		if _, ok := existSet[name]; ok {
			continue
		}
		id := s.NextID()
		if id <= 0 {
			return errors.New("failed to generate id by sid")
		}
		aiModel := &models.AIModel{
			ID:        id,
			Name:      name,
			Type:      models.ModelTypeChat,
			IsEnabled: true,
		}
		simPath := s.priceStore.FindSimilarPath(string(channel.Type), name)
		if simPath != "" {
			aiModel.Extra.SyncModelInfoPath = simPath
			aiModel.Extra.DisableSync = false
			aiModel.Pricing.InputPrice = s.priceStore.GetInputPrice(simPath)
			aiModel.Pricing.OutputPrice = s.priceStore.GetOutputPrice(simPath)
			aiModel.Pricing.CacheReadPrice = s.priceStore.GetCacheReadPrice(simPath)
			aiModel.Pricing.CacheCreatePrice = s.priceStore.GetCacheCreatePrice(simPath)
			aiModel.Pricing.Over200KInputPrice = s.priceStore.GetOver200KInputPrice(simPath)
			aiModel.Pricing.Over200KOutputPrice = s.priceStore.GetOver200KOutputPrice(simPath)
			aiModel.Pricing.Over200KCacheReadPrice = s.priceStore.GetOver200KCacheReadPrice(simPath)
			aiModel.Pricing.Over200KCacheCreatePrice = s.priceStore.GetOver200KCacheWritePrice(simPath)
		}
		toCreate = append(toCreate, aiModel)
	}
	if len(toCreate) == 0 {
		return nil
	}
	return s.modelRepo.BatchCreate(ctx, toCreate)
}
