package service

import (
	"context"
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type ChannelStatsService interface {
	Upsert(ctx context.Context, req v1.UpsertChannelStatsRequest) (*v1.ChannelStatsResponse, error)
	GetByChannelID(ctx context.Context, channelID int64) (*v1.ChannelStatsResponse, error)
	ListByChannelIDs(ctx context.Context, channelIDs []int64) ([]v1.ChannelStatsResponse, error)
}

type channelStatsService struct {
	*Service
	repo repository.ChannelStatsRepo
}

func NewChannelStatsService(base *Service, repo repository.ChannelStatsRepo) ChannelStatsService {
	return &channelStatsService{
		Service: base,
		repo:    repo,
	}
}

func (s *channelStatsService) Upsert(ctx context.Context, req v1.UpsertChannelStatsRequest) (*v1.ChannelStatsResponse, error) {
	if req.ChannelID <= 0 {
		return nil, errors.New("invalid channel id")
	}
	stats := &models.ChannelStats{
		ChannelID:                 req.ChannelID,
		InputToken:                req.InputToken,
		OutputToken:               req.OutputToken,
		CachedCreationInputTokens: req.CachedCreationInputTokens,
		CachedReadInputTokens:     req.CachedReadInputTokens,
		RequestSuccess:            req.RequestSuccess,
		RequestFailed:             req.RequestFailed,
		TotalCostMicros:           req.TotalCostMicros,
		AvgTTFT:                   req.AvgTTFT,
		AvgTPS:                    req.AvgTPS,
	}
	if err := s.repo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetByChannelID(ctx, req.ChannelID)
}

func (s *channelStatsService) GetByChannelID(ctx context.Context, channelID int64) (*v1.ChannelStatsResponse, error) {
	stats, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	resp := toChannelStatsResponse(stats)
	return &resp, nil
}

func (s *channelStatsService) ListByChannelIDs(ctx context.Context, channelIDs []int64) ([]v1.ChannelStatsResponse, error) {
	items, err := s.repo.ListByChannelIDs(ctx, channelIDs)
	if err != nil {
		return nil, err
	}
	resp := make([]v1.ChannelStatsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, toChannelStatsResponse(item))
	}
	return resp, nil
}

func toChannelStatsResponse(stats *models.ChannelStats) v1.ChannelStatsResponse {
	if stats == nil {
		return v1.ChannelStatsResponse{}
	}
	return v1.ChannelStatsResponse{
		ChannelID:                 stats.ChannelID,
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
		AvgTTFT:                   stats.AvgTTFT,
		AvgTPS:                    stats.AvgTPS,
	}
}
