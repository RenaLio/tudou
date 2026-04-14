package service

import (
	"context"
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type TokenStatsService interface {
	Upsert(ctx context.Context, req v1.UpsertTokenStatsRequest) (*v1.TokenStatsResponse, error)
	GetByTokenID(ctx context.Context, tokenID int64) (*v1.TokenStatsResponse, error)
	ListByTokenIDs(ctx context.Context, tokenIDs []int64) ([]v1.TokenStatsResponse, error)
}

type tokenStatsService struct {
	*Service
	repo repository.TokenStatsRepo
}

func NewTokenStatsService(base *Service, repo repository.TokenStatsRepo) TokenStatsService {
	return &tokenStatsService{
		Service: base,
		repo:    repo,
	}
}

func (s *tokenStatsService) Upsert(ctx context.Context, req v1.UpsertTokenStatsRequest) (*v1.TokenStatsResponse, error) {
	if req.TokenID <= 0 {
		return nil, errors.New("invalid token id")
	}
	stats := &models.TokenStats{
		TokenID:                   req.TokenID,
		InputToken:                req.InputToken,
		OutputToken:               req.OutputToken,
		CachedCreationInputTokens: req.CachedCreationInputTokens,
		CachedReadInputTokens:     req.CachedReadInputTokens,
		RequestSuccess:            req.RequestSuccess,
		RequestFailed:             req.RequestFailed,
		TotalCostMicros:           req.TotalCostMicros,
	}
	if err := s.repo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetByTokenID(ctx, req.TokenID)
}

func (s *tokenStatsService) GetByTokenID(ctx context.Context, tokenID int64) (*v1.TokenStatsResponse, error) {
	stats, err := s.repo.GetByTokenID(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	resp := toTokenStatsResponse(stats)
	return &resp, nil
}

func (s *tokenStatsService) ListByTokenIDs(ctx context.Context, tokenIDs []int64) ([]v1.TokenStatsResponse, error) {
	items, err := s.repo.ListByTokenIDs(ctx, tokenIDs)
	if err != nil {
		return nil, err
	}
	resp := make([]v1.TokenStatsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, toTokenStatsResponse(item))
	}
	return resp, nil
}

func toTokenStatsResponse(stats *models.TokenStats) v1.TokenStatsResponse {
	if stats == nil {
		return v1.TokenStatsResponse{}
	}
	return v1.TokenStatsResponse{
		TokenID:                   stats.TokenID,
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
	}
}
