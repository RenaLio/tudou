package handler

import (
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
)

type RequestLogHandler struct {
	*Handler
	requestLogSvc service.RequestLogService
}

func NewRequestLogHandler(base *Handler, requestLogSvc service.RequestLogService) *RequestLogHandler {
	return &RequestLogHandler{
		Handler:       base,
		requestLogSvc: requestLogSvc,
	}
}

func (h *RequestLogHandler) RegisterRoutes(r gin.IRouter) {
	logs := r.Group("/request-log")
	logs.GET("", h.List)
	logs.GET("/:id", h.GetByID)
	logs.DELETE("/:id", h.Delete)
}

func (h *RequestLogHandler) List(ctx *gin.Context) {
	var req v1.ListRequestLogsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}

	opt := repository.RequestLogListOption{
		Page:          req.Page,
		PageSize:      req.PageSize,
		OrderBy:       req.OrderBy,
		Keyword:       req.Keyword,
		RequestID:     req.RequestID,
		UserID:        req.UserID,
		TokenID:       req.TokenID,
		GroupID:       req.GroupID,
		ChannelID:     req.ChannelID,
		Model:         req.Model,
		UpstreamModel: req.UpstreamModel,
		IsStream:      req.IsStream,
	}

	if req.Status != "" {
		status := models.RequestStatus(req.Status)
		opt.Status = &status
	}

	if req.DateFrom != "" {
		if t, err := time.Parse(time.RFC3339, req.DateFrom); err == nil {
			opt.DateFrom = &t
		}
	}
	if req.DateTo != "" {
		if t, err := time.Parse(time.RFC3339, req.DateTo); err == nil {
			opt.DateTo = &t
		}
	}

	items, total, err := h.requestLogSvc.List(ctx.Request.Context(), opt)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}

	respItems := make([]v1.RequestLogResponse, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, toRequestLogResponse(item))
	}

	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	v1.Success(ctx, &v1.ListResponse[v1.RequestLogResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	})
}

func (h *RequestLogHandler) GetByID(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}

	item, err := h.requestLogSvc.GetByID(ctx.Request.Context(), id)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}

	v1.Success(ctx, toRequestLogResponse(item))
}

func (h *RequestLogHandler) Delete(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}

	if err := h.requestLogSvc.Delete(ctx.Request.Context(), id); err != nil {
		HandleServiceError(ctx, err)
		return
	}

	ctx.Status(204)
}

func toRequestLogResponse(log *models.RequestLog) v1.RequestLogResponse {
	if log == nil {
		return v1.RequestLogResponse{}
	}
	return v1.RequestLogResponse{
		ID:                        log.ID,
		RequestID:                 log.RequestID,
		UserID:                    log.UserID,
		TokenID:                   log.TokenID,
		GroupID:                   log.GroupID,
		ChannelID:                 log.ChannelID,
		ChannelName:               log.ChannelName,
		ChannelPriceRate:          log.ChannelPriceRate,
		Model:                     log.Model,
		UpstreamModel:             log.UpstreamModel,
		InputToken:                log.InputToken,
		OutputToken:               log.OutputToken,
		CachedCreationInputTokens: log.CachedCreationInputTokens,
		CachedReadInputTokens:     log.CachedReadInputTokens,
		Pricing:                   log.Pricing,
		CostMicros:                log.CostMicros,
		Status:                    log.Status,
		TTFT:                      log.TTFT,
		TransferTime:              log.TransferTime,
		ErrorCode:                 log.ErrorCode,
		ErrorMsg:                  log.ErrorMsg,
		IsStream:                  log.IsStream,
		Extra:                     log.Extra,
		ProviderDetail:            log.ProviderDetail,
		CreatedAt:                 log.CreatedAt,
	}
}

func normalizePagination(page, pageSize int) (int, int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return page, pageSize, offset
}
