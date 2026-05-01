package handler

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StatsService interface {
	GetChannelStatsByChannelID(ctx context.Context, channelID int64) (*v1.ChannelStatsResponse, error)
	ListAllChannelStats(ctx context.Context) ([]v1.ChannelStatsResponse, error)
	ListChannelModelStatsByChannelID(ctx context.Context, channelID int64) ([]v1.ChannelModelStatsResponse, error)
	GetChannelModelStats(ctx context.Context, channelID int64, model string) (*v1.ChannelModelStatsResponse, error)
	ListAllChannelModelStats(ctx context.Context) ([]v1.ChannelModelStatsResponse, error)
	GetTokenStatsByTokenID(ctx context.Context, tokenID int64) (*v1.TokenStatsResponse, error)
	ListAllTokenStats(ctx context.Context) ([]v1.TokenStatsResponse, error)
	GetUserStatsByUserID(ctx context.Context, userID int64) (*v1.UserStatsResponse, error)
	ListUserUsageDailyStats(ctx context.Context, req v1.ListUserUsageDailyStatsRequest) (*v1.ListResponse[v1.UserUsageDailyStatsResponse], error)
	ListUserUsageHourlyStats(ctx context.Context, req v1.ListUserUsageHourlyStatsRequest) (*v1.ListResponse[v1.UserUsageHourlyStatsResponse], error)
}

var _ StatsService = (*service.StatsService)(nil) // Ensure StatsService is implemented by service.StatsService

type StatsHandler struct {
	*Handler
	svc StatsService
}

func NewStatsHandler(base *Handler, statsService StatsService) *StatsHandler {
	return &StatsHandler{
		Handler: base,
		svc:     statsService,
	}
}

func (h *StatsHandler) RegisterRoutes(r gin.IRouter) {
	stats := r.Group("/stats")

	// 渠道统计（只读）
	stats.GET("/channel/:channelID", h.GetChannelStatsByChannelID)
	stats.GET("/channel", h.ListAllChannelStats)
	stats.GET("/channel/:channelID/model", h.ListChannelModelStatsByChannelID)
	stats.GET("/channel/:channelID/model/:model", h.GetChannelModelStats)
	stats.GET("/channel/model", h.ListAllChannelModelStats)

	// token 统计（只读）
	stats.GET("/token/:tokenID", h.GetTokenStatsByTokenID)
	stats.GET("/token", h.ListTokenStatsByTokenIDs)

	// 用户统计（只读）
	stats.GET("/user/:userID", h.GetUserStatsByUserID)
	stats.GET("/user/usage/daily", h.ListUserUsageDailyStats)
	stats.GET("/user/usage/hourly", h.ListUserUsageHourlyStats)

}

func (h *StatsHandler) GetChannelStatsByChannelID(ctx *gin.Context) {
	channelID, ok := h.ParseIDParam(ctx, "channelID")
	if !ok {
		return
	}
	resp, err := h.svc.GetChannelStatsByChannelID(ctx.Request.Context(), channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HandleNotFound(ctx)
			return
		}
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) ListAllChannelStats(ctx *gin.Context) {
	resp, err := h.svc.ListAllChannelStats(ctx.Request.Context())
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) GetChannelModelStats(ctx *gin.Context) {
	channelID, ok := h.ParseIDParam(ctx, "channelID")
	if !ok {
		return
	}
	model := strings.TrimSpace(ctx.Param("model"))
	if model == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("model is required"), nil)
		return
	}
	resp, err := h.svc.GetChannelModelStats(ctx.Request.Context(), channelID, model)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HandleNotFound(ctx)
			return
		}
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) ListChannelModelStatsByChannelID(ctx *gin.Context) {
	channelID, ok := h.ParseIDParam(ctx, "channelID")
	if !ok {
		return
	}
	resp, err := h.svc.ListChannelModelStatsByChannelID(ctx.Request.Context(), channelID)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}
func (h *StatsHandler) ListAllChannelModelStats(ctx *gin.Context) {
	resp, err := h.svc.ListAllChannelModelStats(ctx.Request.Context())
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) GetTokenStatsByTokenID(ctx *gin.Context) {
	tokenID, ok := h.ParseIDParam(ctx, "tokenID")
	if !ok {
		return
	}
	resp, err := h.svc.GetTokenStatsByTokenID(ctx.Request.Context(), tokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HandleNotFound(ctx)
			return
		}
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) ListTokenStatsByTokenIDs(ctx *gin.Context) {
	resp, err := h.svc.ListAllTokenStats(ctx.Request.Context())
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) GetUserStatsByUserID(ctx *gin.Context) {
	userID, ok := h.ParseIDParam(ctx, "userID")
	if !ok {
		return
	}
	resp, err := h.svc.GetUserStatsByUserID(ctx.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HandleNotFound(ctx)
			return
		}
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) ListUserUsageDailyStats(ctx *gin.Context) {
	var req v1.ListUserUsageDailyStatsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.svc.ListUserUsageDailyStats(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) ListUserUsageHourlyStats(ctx *gin.Context) {
	var req v1.ListUserUsageHourlyStatsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.svc.ListUserUsageHourlyStats(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}
