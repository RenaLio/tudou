package service

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type ChannelModelStatsService interface {
	Upsert(ctx context.Context, req v1.UpsertChannelModelStatsRequest) (*v1.ChannelModelStatsResponse, error)
	GetByChannelModel(ctx context.Context, channelID int64, model string) (*v1.ChannelModelStatsResponse, error)
	ListByChannelID(ctx context.Context, channelID int64) ([]v1.ChannelModelStatsResponse, error)
}

type channelModelStatsService struct {
	*Service
	repo repository.ChannelModelStatsRepo
}

func NewChannelModelStatsService(base *Service, repo repository.ChannelModelStatsRepo) ChannelModelStatsService {
	return &channelModelStatsService{
		Service: base,
		repo:    repo,
	}
}

func (s *channelModelStatsService) Upsert(ctx context.Context, req v1.UpsertChannelModelStatsRequest) (*v1.ChannelModelStatsResponse, error) {
	model := strings.TrimSpace(req.Model)
	if req.ChannelID <= 0 || model == "" {
		return nil, errors.New("invalid channel id or model")
	}
	stats := &models.ChannelModelStats{
		ChannelID:                 req.ChannelID,
		Model:                     model,
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
	return s.GetByChannelModel(ctx, req.ChannelID, model)
}

func (s *channelModelStatsService) GetByChannelModel(ctx context.Context, channelID int64, model string) (*v1.ChannelModelStatsResponse, error) {
	stats, err := s.repo.GetByChannelModel(ctx, channelID, model)
	if err != nil {
		return nil, err
	}
	resp := toChannelModelStatsResponse(stats)
	return &resp, nil
}

func (s *channelModelStatsService) ListByChannelID(ctx context.Context, channelID int64) ([]v1.ChannelModelStatsResponse, error) {
	items, err := s.repo.ListByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	resp := make([]v1.ChannelModelStatsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, toChannelModelStatsResponse(item))
	}
	return resp, nil
}

func toChannelModelStatsResponse(stats *models.ChannelModelStats) v1.ChannelModelStatsResponse {
	if stats == nil {
		return v1.ChannelModelStatsResponse{}
	}
	return v1.ChannelModelStatsResponse{
		ChannelID:                 stats.ChannelID,
		Model:                     stats.Model,
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
