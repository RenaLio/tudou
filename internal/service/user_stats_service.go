package service

import (
	"context"
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type UserStatsService interface {
	Upsert(ctx context.Context, req v1.UpsertUserStatsRequest) (*v1.UserStatsResponse, error)
	GetByUserID(ctx context.Context, userID int64) (*v1.UserStatsResponse, error)
	ListByUserIDs(ctx context.Context, userIDs []int64) ([]v1.UserStatsResponse, error)
}

type userStatsService struct {
	*Service
	repo repository.UserStatsRepo
}

func NewUserStatsService(base *Service, repo repository.UserStatsRepo) UserStatsService {
	return &userStatsService{
		Service: base,
		repo:    repo,
	}
}

func (s *userStatsService) Upsert(ctx context.Context, req v1.UpsertUserStatsRequest) (*v1.UserStatsResponse, error) {
	if req.UserID <= 0 {
		return nil, errors.New("invalid user id")
	}
	stats := &models.UserStats{
		UserID:                    req.UserID,
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
	return s.GetByUserID(ctx, req.UserID)
}

func (s *userStatsService) GetByUserID(ctx context.Context, userID int64) (*v1.UserStatsResponse, error) {
	stats, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := toUserStatsResponse(stats)
	return &resp, nil
}

func (s *userStatsService) ListByUserIDs(ctx context.Context, userIDs []int64) ([]v1.UserStatsResponse, error) {
	items, err := s.repo.ListByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	resp := make([]v1.UserStatsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, toUserStatsResponse(item))
	}
	return resp, nil
}

func toUserStatsResponse(stats *models.UserStats) v1.UserStatsResponse {
	if stats == nil {
		return v1.UserStatsResponse{}
	}
	return v1.UserStatsResponse{
		UserID:                    stats.UserID,
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
	}
}
