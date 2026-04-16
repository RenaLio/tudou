package handler

import (
	"errors"
	"strconv"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StatsHandler struct {
	*Handler
	StatsService service.StatsService
}

func NewStatsHandler(base *Handler, statsService service.StatsService) *StatsHandler {
	return &StatsHandler{
		Handler:      base,
		StatsService: statsService,
	}
}

func (h *StatsHandler) RegisterRoutes(r gin.IRouter) {
	stats := r.Group("/stats")

	// 渠道统计（只读）
	stats.GET("/channel/:channelID", h.GetChannelStatsByChannelID)
	stats.GET("/channel", h.ListChannelStatsByChannelIDs)
	stats.GET("/channel/:channelID/model", h.ListChannelModelStatsByChannelID)
	stats.GET("/channel/:channelID/model/:model", h.GetChannelModelStats)

	// token 统计（只读）
	stats.GET("/token/:tokenID", h.GetTokenStatsByTokenID)
	stats.GET("/token", h.ListTokenStatsByTokenIDs)

	// 用户统计（只读）
	stats.GET("/user/:userID", h.GetUserStatsByUserID)
	stats.GET("/user", h.ListUserStatsByUserIDs)
	stats.GET("/user/usage/daily", h.ListUserUsageDailyStats)
	stats.GET("/user/:userID/usage/daily/:date", h.GetUserUsageDailyStatsByUserDate)
	stats.GET("/user/usage/hourly", h.ListUserUsageHourlyStats)
	stats.GET("/user/:userID/usage/hourly/:date/:hour", h.GetUserUsageHourlyStatsByUserDateHour)
}

func (h *StatsHandler) GetChannelStatsByChannelID(ctx *gin.Context) {
	channelID, ok := h.ParseIDParam(ctx, "channelID")
	if !ok {
		return
	}
	resp, err := h.StatsService.GetChannelStatsByChannelID(ctx.Request.Context(), channelID)
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

func (h *StatsHandler) ListChannelStatsByChannelIDs(ctx *gin.Context) {
	ids, ok := parseIDListQuery(ctx, "ids")
	if !ok {
		return
	}
	resp, err := h.StatsService.ListChannelStatsByChannelIDs(ctx.Request.Context(), ids)
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
	resp, err := h.StatsService.GetChannelModelStats(ctx.Request.Context(), channelID, model)
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
	resp, err := h.StatsService.ListChannelModelStatsByChannelID(ctx.Request.Context(), channelID)
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
	resp, err := h.StatsService.GetTokenStatsByTokenID(ctx.Request.Context(), tokenID)
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
	ids, ok := parseIDListQuery(ctx, "ids")
	if !ok {
		return
	}
	resp, err := h.StatsService.ListTokenStatsByTokenIDs(ctx.Request.Context(), ids)
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
	resp, err := h.StatsService.GetUserStatsByUserID(ctx.Request.Context(), userID)
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

func (h *StatsHandler) ListUserStatsByUserIDs(ctx *gin.Context) {
	ids, ok := parseIDListQuery(ctx, "ids")
	if !ok {
		return
	}
	resp, err := h.StatsService.ListUserStatsByUserIDs(ctx.Request.Context(), ids)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) GetUserUsageDailyStatsByUserDate(ctx *gin.Context) {
	userID, ok := h.ParseIDParam(ctx, "userID")
	if !ok {
		return
	}
	date := strings.TrimSpace(ctx.Param("date"))
	if date == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("date is required"), nil)
		return
	}
	resp, err := h.StatsService.GetUserUsageDailyStatsByUserDate(ctx.Request.Context(), userID, date)
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
	resp, err := h.StatsService.ListUserUsageDailyStats(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *StatsHandler) GetUserUsageHourlyStatsByUserDateHour(ctx *gin.Context) {
	userID, ok := h.ParseIDParam(ctx, "userID")
	if !ok {
		return
	}
	date := strings.TrimSpace(ctx.Param("date"))
	if date == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("date is required"), nil)
		return
	}
	hour, err := strconv.Atoi(strings.TrimSpace(ctx.Param("hour")))
	if err != nil || hour < 0 || hour > 23 {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("invalid hour"), nil)
		return
	}
	resp, err := h.StatsService.GetUserUsageHourlyStatsByUserDateHour(ctx.Request.Context(), userID, date, hour)
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

func (h *StatsHandler) ListUserUsageHourlyStats(ctx *gin.Context) {
	var req v1.ListUserUsageHourlyStatsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.StatsService.ListUserUsageHourlyStats(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func parseIDListQuery(ctx *gin.Context, key string) ([]int64, bool) {
	raw := strings.TrimSpace(ctx.Query(key))
	if raw == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage(key+" is required"), nil)
		return nil, false
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			v1.Fail(ctx, v1.ErrBadRequest.WithMessage("invalid ids"), nil)
			return nil, false
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("invalid ids"), nil)
		return nil, false
	}
	return ids, true
}
