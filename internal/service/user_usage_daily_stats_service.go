package service

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

type UserUsageDailyStatsService interface {
	Upsert(ctx context.Context, req v1.UpsertUserUsageDailyStatsRequest) (*v1.UserUsageDailyStatsResponse, error)
	GetByUserDate(ctx context.Context, userID int64, date string) (*v1.UserUsageDailyStatsResponse, error)
	List(ctx context.Context, req v1.ListUserUsageDailyStatsRequest) (*v1.ListResponse[v1.UserUsageDailyStatsResponse], error)
}

type userUsageDailyStatsService struct {
	*Service
	repo repository.UserUsageDailyStatsRepo
}

func NewUserUsageDailyStatsService(base *Service, repo repository.UserUsageDailyStatsRepo) UserUsageDailyStatsService {
	return &userUsageDailyStatsService{
		Service: base,
		repo:    repo,
	}
}

func (s *userUsageDailyStatsService) Upsert(ctx context.Context, req v1.UpsertUserUsageDailyStatsRequest) (*v1.UserUsageDailyStatsResponse, error) {
	date := strings.TrimSpace(req.Date)
	if req.UserID <= 0 || date == "" {
		return nil, errors.New("invalid user id or date")
	}

	// 先尝试读取已有记录；不存在时创建新 ID。
	stats, err := s.repo.GetByUserDate(ctx, req.UserID, date)
	if err != nil {
		stats = &models.UserUsageDailyStats{
			ID:     s.NextID(),
			UserID: req.UserID,
			Date:   date,
		}
	}
	if stats.ID <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}

	stats.InputToken = req.InputToken
	stats.OutputToken = req.OutputToken
	stats.CachedCreationInputTokens = req.CachedCreationInputTokens
	stats.CachedReadInputTokens = req.CachedReadInputTokens
	stats.RequestSuccess = req.RequestSuccess
	stats.RequestFailed = req.RequestFailed
	stats.TotalCostMicros = req.TotalCostMicros

	if err = s.repo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetByUserDate(ctx, req.UserID, date)
}

func (s *userUsageDailyStatsService) GetByUserDate(ctx context.Context, userID int64, date string) (*v1.UserUsageDailyStatsResponse, error) {
	stats, err := s.repo.GetByUserDate(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	resp := toUserUsageDailyStatsResponse(stats)
	return &resp, nil
}

func (s *userUsageDailyStatsService) List(ctx context.Context, req v1.ListUserUsageDailyStatsRequest) (*v1.ListResponse[v1.UserUsageDailyStatsResponse], error) {
	opt := repository.UserUsageDailyStatsListOption{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		UserID:   req.UserID,
		DateFrom: req.DateFrom,
		DateTo:   req.DateTo,
	}
	items, total, err := s.repo.List(ctx, opt)
	if err != nil {
		return nil, err
	}
	respItems := make([]v1.UserUsageDailyStatsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toUserUsageDailyStatsResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.UserUsageDailyStatsResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func toUserUsageDailyStatsResponse(stats *models.UserUsageDailyStats) v1.UserUsageDailyStatsResponse {
	if stats == nil {
		return v1.UserUsageDailyStatsResponse{}
	}
	return v1.UserUsageDailyStatsResponse{
		ID:                        stats.ID,
		UserID:                    stats.UserID,
		Date:                      stats.Date,
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
		CreatedAt:                 stats.CreatedAt,
		UpdatedAt:                 stats.UpdatedAt,
	}
}
