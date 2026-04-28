package service

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
)

//type StatsService interface {
//	UpsertChannelStats(ctx context.Context, req v1.UpsertChannelStatsRequest) (*v1.ChannelStatsResponse, error)
//	GetChannelStatsByChannelID(ctx context.Context, channelID int64) (*v1.ChannelStatsResponse, error)
//	ListChannelStatsByChannelIDs(ctx context.Context, channelIDs []int64) ([]v1.ChannelStatsResponse, error)
//
//	UpsertChannelModelStats(ctx context.Context, req v1.UpsertChannelModelStatsRequest) (*v1.ChannelModelStatsResponse, error)
//	GetChannelModelStats(ctx context.Context, channelID int64, model string) (*v1.ChannelModelStatsResponse, error)
//	ListChannelModelStatsByChannelID(ctx context.Context, channelID int64) ([]v1.ChannelModelStatsResponse, error)
//
//	UpsertTokenStats(ctx context.Context, req v1.UpsertTokenStatsRequest) (*v1.TokenStatsResponse, error)
//	GetTokenStatsByTokenID(ctx context.Context, tokenID int64) (*v1.TokenStatsResponse, error)
//	ListTokenStatsByTokenIDs(ctx context.Context, tokenIDs []int64) ([]v1.TokenStatsResponse, error)
//
//	UpsertUserStats(ctx context.Context, req v1.UpsertUserStatsRequest) (*v1.UserStatsResponse, error)
//	GetUserStatsByUserID(ctx context.Context, userID int64) (*v1.UserStatsResponse, error)
//	ListUserStatsByUserIDs(ctx context.Context, userIDs []int64) ([]v1.UserStatsResponse, error)
//
//	UpsertUserUsageDailyStats(ctx context.Context, req v1.UpsertUserUsageDailyStatsRequest) (*v1.UserUsageDailyStatsResponse, error)
//	GetUserUsageDailyStatsByUserDate(ctx context.Context, userID int64, date string) (*v1.UserUsageDailyStatsResponse, error)
//	ListUserUsageDailyStats(ctx context.Context, req v1.ListUserUsageDailyStatsRequest) (*v1.ListResponse[v1.UserUsageDailyStatsResponse], error)
//
//	UpsertUserUsageHourlyStats(ctx context.Context, req v1.UpsertUserUsageHourlyStatsRequest) (*v1.UserUsageHourlyStatsResponse, error)
//	GetUserUsageHourlyStatsByUserDateHour(ctx context.Context, userID int64, date string, hour int) (*v1.UserUsageHourlyStatsResponse, error)
//	ListUserUsageHourlyStats(ctx context.Context, req v1.ListUserUsageHourlyStatsRequest) (*v1.ListResponse[v1.UserUsageHourlyStatsResponse], error)
//}

type StatsService struct {
	*Service
	channelStatsRepo         repository.ChannelStatsRepo
	channelModelStatsRepo    repository.ChannelModelStatsRepo
	tokenStatsRepo           repository.TokenStatsRepo
	userStatsRepo            repository.UserStatsRepo
	userUsageDailyStatsRepo  repository.UserUsageDailyStatsRepo
	userUsageHourlyStatsRepo repository.UserUsageHourlyStatsRepo
}

func (s *StatsService) ListAllTokenStats(ctx context.Context) ([]v1.TokenStatsResponse, error) {
	items, err := s.tokenStatsRepo.ListAll(ctx)
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

func NewStatsService(
	base *Service,
	channelStatsRepo repository.ChannelStatsRepo,
	channelModelStatsRepo repository.ChannelModelStatsRepo,
	tokenStatsRepo repository.TokenStatsRepo,
	userStatsRepo repository.UserStatsRepo,
	userUsageDailyStatsRepo repository.UserUsageDailyStatsRepo,
	userUsageHourlyStatsRepo repository.UserUsageHourlyStatsRepo,
) *StatsService {
	return &StatsService{
		Service:                  base,
		channelStatsRepo:         channelStatsRepo,
		channelModelStatsRepo:    channelModelStatsRepo,
		tokenStatsRepo:           tokenStatsRepo,
		userStatsRepo:            userStatsRepo,
		userUsageDailyStatsRepo:  userUsageDailyStatsRepo,
		userUsageHourlyStatsRepo: userUsageHourlyStatsRepo,
	}
}

func (s *StatsService) ListAllChannelStats(ctx context.Context) ([]v1.ChannelStatsResponse, error) {
	items, err := s.channelStatsRepo.ListAll(ctx)
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

func (s *StatsService) UpsertChannelStats(ctx context.Context, req v1.UpsertChannelStatsRequest) (*v1.ChannelStatsResponse, error) {
	if req.ChannelID <= 0 {
		return nil, errors.New("invalid channel id")
	}
	stats := &models.ChannelStats{
		ChannelID:                 req.ChannelID,
		ChannelName:               strings.TrimSpace(req.ChannelName),
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
	if err := s.channelStatsRepo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetChannelStatsByChannelID(ctx, req.ChannelID)
}

func (s *StatsService) GetChannelStatsByChannelID(ctx context.Context, channelID int64) (*v1.ChannelStatsResponse, error) {
	stats, err := s.channelStatsRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	resp := toChannelStatsResponse(stats)
	return &resp, nil
}

func (s *StatsService) ListChannelStatsByChannelIDs(ctx context.Context, channelIDs []int64) ([]v1.ChannelStatsResponse, error) {
	items, err := s.channelStatsRepo.ListByChannelIDs(ctx, channelIDs)
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

func (s *StatsService) UpsertChannelModelStats(ctx context.Context, req v1.UpsertChannelModelStatsRequest) (*v1.ChannelModelStatsResponse, error) {
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
	if err := s.channelModelStatsRepo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetChannelModelStats(ctx, req.ChannelID, model)
}

func (s *StatsService) GetChannelModelStats(ctx context.Context, channelID int64, model string) (*v1.ChannelModelStatsResponse, error) {
	model = strings.TrimSpace(model)
	stats, err := s.channelModelStatsRepo.GetByChannelModel(ctx, channelID, model)
	if err != nil {
		return nil, err
	}
	resp := toChannelModelStatsResponse(stats)
	return &resp, nil
}

func (s *StatsService) ListChannelModelStatsByChannelID(ctx context.Context, channelID int64) ([]v1.ChannelModelStatsResponse, error) {
	items, err := s.channelModelStatsRepo.ListByChannelID(ctx, channelID)
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

func (s *StatsService) UpsertTokenStats(ctx context.Context, req v1.UpsertTokenStatsRequest) (*v1.TokenStatsResponse, error) {
	if req.TokenID <= 0 {
		return nil, errors.New("invalid token id")
	}
	stats := &models.TokenStats{
		TokenID:                   req.TokenID,
		TokenName:                 strings.TrimSpace(req.TokenName),
		InputToken:                req.InputToken,
		OutputToken:               req.OutputToken,
		CachedCreationInputTokens: req.CachedCreationInputTokens,
		CachedReadInputTokens:     req.CachedReadInputTokens,
		RequestSuccess:            req.RequestSuccess,
		RequestFailed:             req.RequestFailed,
		TotalCostMicros:           req.TotalCostMicros,
	}
	if err := s.tokenStatsRepo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetTokenStatsByTokenID(ctx, req.TokenID)
}

func (s *StatsService) GetTokenStatsByTokenID(ctx context.Context, tokenID int64) (*v1.TokenStatsResponse, error) {
	stats, err := s.tokenStatsRepo.GetByTokenID(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	resp := toTokenStatsResponse(stats)
	return &resp, nil
}

func (s *StatsService) ListTokenStatsByTokenIDs(ctx context.Context, tokenIDs []int64) ([]v1.TokenStatsResponse, error) {
	items, err := s.tokenStatsRepo.ListByTokenIDs(ctx, tokenIDs)
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

func (s *StatsService) UpsertUserStats(ctx context.Context, req v1.UpsertUserStatsRequest) (*v1.UserStatsResponse, error) {
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
	if err := s.userStatsRepo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetUserStatsByUserID(ctx, req.UserID)
}

func (s *StatsService) GetUserStatsByUserID(ctx context.Context, userID int64) (*v1.UserStatsResponse, error) {
	stats, err := s.userStatsRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := toUserStatsResponse(stats)
	return &resp, nil
}

func (s *StatsService) ListUserStatsByUserIDs(ctx context.Context, userIDs []int64) ([]v1.UserStatsResponse, error) {
	items, err := s.userStatsRepo.ListByUserIDs(ctx, userIDs)
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

func (s *StatsService) UpsertUserUsageDailyStats(ctx context.Context, req v1.UpsertUserUsageDailyStatsRequest) (*v1.UserUsageDailyStatsResponse, error) {
	if req.UserID <= 0 || req.Date.IsZero() {
		return nil, errors.New("invalid user id or date")
	}
	date := dateToKey(req.Date)

	stats, err := s.userUsageDailyStatsRepo.GetByUserDate(ctx, req.UserID, date)
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

	if err = s.userUsageDailyStatsRepo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetUserUsageDailyStatsByUserDate(ctx, req.UserID, date)
}

func (s *StatsService) GetUserUsageDailyStatsByUserDate(ctx context.Context, userID int64, date string) (*v1.UserUsageDailyStatsResponse, error) {
	date = strings.TrimSpace(date)
	if userID <= 0 || date == "" {
		return nil, errors.New("invalid user id or date")
	}
	stats, err := s.userUsageDailyStatsRepo.GetByUserDate(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	resp := toUserUsageDailyStatsResponse(stats)
	return &resp, nil
}

func (s *StatsService) ListUserUsageDailyStats(ctx context.Context, req v1.ListUserUsageDailyStatsRequest) (*v1.ListResponse[v1.UserUsageDailyStatsResponse], error) {
	dateFrom := ""
	if req.StartTime != nil && !req.StartTime.IsZero() {
		dateFrom = dateToKey(*req.StartTime)
	}
	dateTo := ""
	if req.EndTime != nil && !req.EndTime.IsZero() {
		dateTo = dateToKey(*req.EndTime)
	}
	opt := repository.UserUsageDailyStatsListOption{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		UserID:   req.UserID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
	}
	items, total, err := s.userUsageDailyStatsRepo.List(ctx, opt)
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

func (s *StatsService) UpsertUserUsageHourlyStats(ctx context.Context, req v1.UpsertUserUsageHourlyStatsRequest) (*v1.UserUsageHourlyStatsResponse, error) {
	if req.UserID <= 0 || req.Date.IsZero() || req.Hour < 0 || req.Hour > 23 {
		return nil, errors.New("invalid user id/date/hour")
	}
	date := dateToKey(req.Date)

	stats, err := s.userUsageHourlyStatsRepo.GetByUserDateHour(ctx, req.UserID, date, req.Hour)
	if err != nil {
		stats = &models.UserUsageHourlyStats{
			ID:     s.NextID(),
			UserID: req.UserID,
			Date:   date,
			Hour:   req.Hour,
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

	if err = s.userUsageHourlyStatsRepo.Upsert(ctx, stats); err != nil {
		return nil, err
	}
	return s.GetUserUsageHourlyStatsByUserDateHour(ctx, req.UserID, date, req.Hour)
}

func (s *StatsService) GetUserUsageHourlyStatsByUserDateHour(ctx context.Context, userID int64, date string, hour int) (*v1.UserUsageHourlyStatsResponse, error) {
	date = strings.TrimSpace(date)
	if userID <= 0 || date == "" || hour < 0 || hour > 23 {
		return nil, errors.New("invalid user id/date/hour")
	}
	stats, err := s.userUsageHourlyStatsRepo.GetByUserDateHour(ctx, userID, date, hour)
	if err != nil {
		return nil, err
	}
	resp := toUserUsageHourlyStatsResponse(stats)
	return &resp, nil
}

func (s *StatsService) ListUserUsageHourlyStats(ctx context.Context, req v1.ListUserUsageHourlyStatsRequest) (*v1.ListResponse[v1.UserUsageHourlyStatsResponse], error) {
	dateFrom := ""
	hourFrom := 0
	if req.StartTime != nil && !req.StartTime.IsZero() {
		dateFrom = dateToKey(*req.StartTime)
		hourFrom = req.StartTime.Hour()
	}
	dateTo := ""
	hourTo := 0
	if req.EndTime != nil && !req.EndTime.IsZero() {
		dateTo = dateToKey(*req.EndTime)
		hourTo = req.EndTime.Hour()
	}
	opt := repository.UserUsageHourlyStatsListOption{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		UserID:   req.UserID,
		DateFrom: dateFrom,
		HourFrom: hourFrom,
		DateTo:   dateTo,
		HourTo:   hourTo,
	}
	items, total, err := s.userUsageHourlyStatsRepo.List(ctx, opt)
	if err != nil {
		return nil, err
	}
	respItems := make([]v1.UserUsageHourlyStatsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toUserUsageHourlyStatsResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.UserUsageHourlyStatsResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func toChannelStatsResponse(stats *models.ChannelStats) v1.ChannelStatsResponse {
	if stats == nil {
		return v1.ChannelStatsResponse{}
	}
	return v1.ChannelStatsResponse{
		ChannelID:                 stats.ChannelID,
		ChannelName:               stats.ChannelName,
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
		TotalCost:                 models.MoneyMicrosToFloat(stats.TotalCostMicros),
		AvgTTFT:                   stats.AvgTTFT,
		AvgTPS:                    stats.AvgTPS,
		Window3H:                  toObservationWindow3HResponse(stats.Window3H),
	}
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
		TotalCost:                 models.MoneyMicrosToFloat(stats.TotalCostMicros),
		AvgTTFT:                   stats.AvgTTFT,
		AvgTPS:                    stats.AvgTPS,
		Window3H:                  toObservationWindow3HResponse(stats.Window3H),
	}
}

func toTokenStatsResponse(stats *models.TokenStats) v1.TokenStatsResponse {
	if stats == nil {
		return v1.TokenStatsResponse{}
	}
	return v1.TokenStatsResponse{
		TokenID:                   stats.TokenID,
		TokenName:                 stats.TokenName,
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
		TotalCost:                 models.MoneyMicrosToFloat(stats.TotalCostMicros),
	}
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
		TotalCost:                 models.MoneyMicrosToFloat(stats.TotalCostMicros),
	}
}

func toUserUsageDailyStatsResponse(stats *models.UserUsageDailyStats) v1.UserUsageDailyStatsResponse {
	if stats == nil {
		return v1.UserUsageDailyStatsResponse{}
	}
	return v1.UserUsageDailyStatsResponse{
		ID:                        stats.ID,
		UserID:                    stats.UserID,
		Date:                      parseDateKey(stats.Date),
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
		TotalCost:                 models.MoneyMicrosToFloat(stats.TotalCostMicros),
		CreatedAt:                 stats.CreatedAt,
		UpdatedAt:                 stats.UpdatedAt,
	}
}

func toUserUsageHourlyStatsResponse(stats *models.UserUsageHourlyStats) v1.UserUsageHourlyStatsResponse {
	if stats == nil {
		return v1.UserUsageHourlyStatsResponse{}
	}
	return v1.UserUsageHourlyStatsResponse{
		ID:                        stats.ID,
		UserID:                    stats.UserID,
		Date:                      parseDateKey(stats.Date),
		Hour:                      stats.Hour,
		InputToken:                stats.InputToken,
		OutputToken:               stats.OutputToken,
		CachedCreationInputTokens: stats.CachedCreationInputTokens,
		CachedReadInputTokens:     stats.CachedReadInputTokens,
		RequestSuccess:            stats.RequestSuccess,
		RequestFailed:             stats.RequestFailed,
		TotalCostMicros:           stats.TotalCostMicros,
		TotalCost:                 models.MoneyMicrosToFloat(stats.TotalCostMicros),
		CreatedAt:                 stats.CreatedAt,
		UpdatedAt:                 stats.UpdatedAt,
	}
}

func dateToKey(value time.Time) string {
	return value.Format("2006-01-02")
}

func parseDateKey(value string) time.Time {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return date
}
