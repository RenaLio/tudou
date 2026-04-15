package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type TokenService interface {
	Create(ctx context.Context, req v1.CreateTokenRequest) (*v1.TokenResponse, error)
	BatchCreate(ctx context.Context, reqs []v1.CreateTokenRequest) ([]v1.TokenResponse, error)
	GetByID(ctx context.Context, id int64, withRelations bool) (*v1.TokenWithRelationsResponse, error)
	GetByToken(ctx context.Context, token string, withRelations bool) (*v1.TokenWithRelationsResponse, error)
	GetAvailableByToken(ctx context.Context, token string) (*v1.TokenWithRelationsResponse, error)
	List(ctx context.Context, req v1.ListTokensRequest) (*v1.ListResponse[v1.TokenWithRelationsResponse], error)
	Update(ctx context.Context, id int64, req v1.UpdateTokenRequest) (*v1.TokenWithRelationsResponse, error)
	UpdateStatus(ctx context.Context, id int64, req v1.SetTokenStatusRequest) (*v1.TokenWithRelationsResponse, error)
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

type tokenService struct {
	*Service
	repo repository.TokenRepo
}

func NewTokenService(base *Service, repo repository.TokenRepo) TokenService {
	return &tokenService{
		Service: base,
		repo:    repo,
	}
}

func (s *tokenService) Create(ctx context.Context, req v1.CreateTokenRequest) (*v1.TokenResponse, error) {
	token, err := s.buildTokenByCreateReq(req)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, token); err != nil {
		return nil, err
	}
	resp := toTokenResponse(token)
	return &resp, nil
}

func (s *tokenService) BatchCreate(ctx context.Context, reqs []v1.CreateTokenRequest) ([]v1.TokenResponse, error) {
	if len(reqs) == 0 {
		return []v1.TokenResponse{}, nil
	}
	tokens := make([]*models.Token, 0, len(reqs))
	for _, req := range reqs {
		token, err := s.buildTokenByCreateReq(req)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := s.repo.BatchCreate(ctx, tokens); err != nil {
		return nil, err
	}

	resp := make([]v1.TokenResponse, 0, len(tokens))
	for _, token := range tokens {
		if token == nil {
			continue
		}
		resp = append(resp, toTokenResponse(token))
	}
	return resp, nil
}

func (s *tokenService) GetByID(ctx context.Context, id int64, withRelations bool) (*v1.TokenWithRelationsResponse, error) {
	var (
		token *models.Token
		err   error
	)
	if withRelations {
		token, err = s.repo.GetByIDWithRelations(ctx, id)
	} else {
		token, err = s.repo.GetByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	resp := toTokenWithRelationsResponse(token)
	return &resp, nil
}

func (s *tokenService) GetByToken(ctx context.Context, token string, withRelations bool) (*v1.TokenWithRelationsResponse, error) {
	var (
		data *models.Token
		err  error
	)
	if withRelations {
		data, err = s.repo.GetByTokenWithRelations(ctx, token)
	} else {
		data, err = s.repo.GetByToken(ctx, token)
	}
	if err != nil {
		return nil, err
	}
	resp := toTokenWithRelationsResponse(data)
	return &resp, nil
}

func (s *tokenService) GetAvailableByToken(ctx context.Context, token string) (*v1.TokenWithRelationsResponse, error) {
	data, err := s.repo.GetAvailableByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	resp := toTokenWithRelationsResponse(data)
	return &resp, nil
}

func (s *tokenService) List(ctx context.Context, req v1.ListTokensRequest) (*v1.ListResponse[v1.TokenWithRelationsResponse], error) {
	opt := repository.TokenListOption{
		Page:          req.Page,
		PageSize:      req.PageSize,
		OrderBy:       req.OrderBy,
		Keyword:       req.Keyword,
		UserID:        req.UserID,
		GroupID:       req.GroupID,
		OnlyAvailable: req.OnlyAvailable,
		PreloadUser:   req.PreloadUser,
		PreloadGroup:  req.PreloadGroup,
		PreloadStats:  req.PreloadStats,
	}
	if req.Status != "" {
		status := models.TokenStatus(req.Status)
		opt.Status = &status
	}

	items, total, err := s.repo.List(ctx, opt)
	if err != nil {
		return nil, err
	}
	respItems := make([]v1.TokenWithRelationsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toTokenWithRelationsResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.TokenWithRelationsResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func (s *tokenService) Update(ctx context.Context, id int64, req v1.UpdateTokenRequest) (*v1.TokenWithRelationsResponse, error) {
	token, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}
	patchTokenByUpdateReq(token, req)
	if err = s.repo.Update(ctx, token); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id, true)
}

func (s *tokenService) UpdateStatus(ctx context.Context, id int64, req v1.SetTokenStatusRequest) (*v1.TokenWithRelationsResponse, error) {
	if err := s.repo.UpdateStatus(ctx, id, req.Status); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id, true)
}

func (s *tokenService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *tokenService) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}

func (s *tokenService) buildTokenByCreateReq(req v1.CreateTokenRequest) (*models.Token, error) {
	tokenValue := GenToken(req.UserID, req.GroupID)
	if req.UserID <= 0 || req.GroupID <= 0 || tokenValue == "" {
		return nil, errors.New("userID/groupID are required")
	}
	id := s.NextID()
	if id <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}
	token := &models.Token{
		ID:        id,
		UserID:    req.UserID,
		GroupID:   req.GroupID,
		Token:     tokenValue,
		Name:      strings.TrimSpace(req.Name),
		ExpiresAt: req.ExpiresAt,
		Settings:  req.Settings,
	}
	if req.Status == nil {
		token.Status = models.TokenStatusEnabled
	} else {
		token.Status = *req.Status
	}
	if req.Limit == nil {
		token.LimitMicros = -1
	} else {
		token.LimitMicros = int64((*req.Limit) * float64(models.GetMoneyMicrosPerUnit()))
	}
	if req.LoadBalanceStrategy == nil {
		token.LoadBalanceStrategy = models.LoadBalanceStrategyPerformance
	} else {
		token.LoadBalanceStrategy = *req.LoadBalanceStrategy
	}
	return token, nil
}

func patchTokenByUpdateReq(token *models.Token, req v1.UpdateTokenRequest) {
	if token == nil {
		return
	}
	if req.Name != nil {
		token.Name = strings.TrimSpace(*req.Name)
	}
	if req.Status != nil {
		token.Status = *req.Status
	}
	if req.LimitMicros != nil {
		token.LimitMicros = *req.LimitMicros
	}
	if req.ExpiresAt != nil {
		token.ExpiresAt = *req.ExpiresAt
	}
	if req.LoadBalanceStrategy != nil {
		token.LoadBalanceStrategy = *req.LoadBalanceStrategy
	}
	if req.Settings != nil {
		token.Settings = *req.Settings
	}
}

func toTokenResponse(token *models.Token) v1.TokenResponse {
	if token == nil {
		return v1.TokenResponse{}
	}
	return v1.TokenResponse{
		ID:                  token.ID,
		UserID:              token.UserID,
		GroupID:             token.GroupID,
		Token:               token.Token,
		Name:                token.Name,
		Status:              token.Status,
		LimitMicros:         token.LimitMicros,
		ExpiresAt:           token.ExpiresAt,
		LoadBalanceStrategy: token.LoadBalanceStrategy,
		Settings:            token.Settings,
		CreatedAt:           token.CreatedAt,
		UpdatedAt:           token.UpdatedAt,
	}
}

func toTokenWithRelationsResponse(token *models.Token) v1.TokenWithRelationsResponse {
	if token == nil {
		return v1.TokenWithRelationsResponse{}
	}
	resp := v1.TokenWithRelationsResponse{
		TokenResponse: toTokenResponse(token),
	}

	// 只有预加载过关系时这些字段才有值
	if token.User.ID != 0 {
		userResp := v1.UserResponse{
			ID:          token.User.ID,
			Username:    token.User.Username,
			Email:       token.User.Email,
			Phone:       token.User.Phone,
			Nickname:    token.User.Nickname,
			Avatar:      token.User.Avatar,
			Status:      token.User.Status,
			Role:        token.User.Role,
			LastLoginAt: token.User.LastLoginAt,
			LastLoginIP: token.User.LastLoginIP,
			LoginCount:  token.User.LoginCount,
			Settings:    token.User.Settings,
			CreatedAt:   token.User.CreatedAt,
			UpdatedAt:   token.User.UpdatedAt,
		}
		resp.User = &userResp
	}

	if token.Group.ID != 0 {
		channelIDs := make([]int64, 0, len(token.Group.Channels))
		for _, channel := range token.Group.Channels {
			channelIDs = append(channelIDs, channel.ID)
		}
		groupResp := v1.ChannelGroupResponse{
			ID:                  token.Group.ID,
			Name:                token.Group.Name,
			NameRemark:          token.Group.NameRemark,
			Description:         token.Group.Description,
			PermissionNum:       token.Group.PermissionNum,
			LoadBalanceStrategy: token.Group.LoadBalanceStrategy,
			CreatedAt:           token.Group.CreatedAt,
			UpdatedAt:           token.Group.UpdatedAt,
			ChannelIDs:          channelIDs,
		}
		resp.Group = &groupResp
	}

	if token.Stats.TokenID != 0 {
		stats := token.Stats
		resp.Stats = &stats
	}
	return resp
}

func GenToken(userId int64, groupId int64) string {

	return RandomStringId(32)
}

func RandomStringId(length int) string {

	//const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, length)

	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}

	return string(b)
}
