package handler

import (
	"errors"
	"net/http"
	"strconv"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChannelHandler struct {
	*Handler
	ChannelService service.ChannelService
	RelayService   RelayService
}

func NewChannelHandler(
	base *Handler,
	channelService service.ChannelService,
	relayService RelayService,
) *ChannelHandler {
	return &ChannelHandler{
		Handler:        base,
		ChannelService: channelService,
		RelayService:   relayService,
	}
}

func (h *ChannelHandler) RegisterRoutes(r gin.IRouter) {
	channels := r.Group("/channel")
	channels.POST("/fetch-model", h.FetchChannelRelays)
	channels.POST("", h.CreateChannel)
	channels.GET("", h.ListChannels)
	channels.GET("/:id", h.GetChannelByID)
	channels.PUT("/:id", h.UpdateChannel)
	channels.PATCH("/:id/status", h.SetChannelStatus)
	channels.PUT("/:id/groups", h.ReplaceChannelGroups)
	channels.DELETE("/:id", h.DeleteChannel)
}

func (h *ChannelHandler) CreateChannel(ctx *gin.Context) {
	var req v1.CreateChannelRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	// 转换 GroupStringIDs 到 GroupIDs
	if len(req.GroupStringIDs) > 0 {
		req.GroupIDs = make([]int64, 0, len(req.GroupStringIDs))
		for _, idStr := range req.GroupStringIDs {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				req.GroupIDs = append(req.GroupIDs, id)
			}
		}
	}
	resp, err := h.ChannelService.Create(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ChannelHandler) ListChannels(ctx *gin.Context) {
	var req v1.ListChannelsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	// 转换 GroupStringID 到 GroupID
	if req.GroupStringID != "" {
		if id, err := strconv.ParseInt(req.GroupStringID, 10, 64); err == nil {
			req.GroupID = id
		}
	}
	resp, err := h.ChannelService.List(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ChannelHandler) GetChannelByID(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	resp, err := h.ChannelService.GetByID(ctx.Request.Context(), id, true)
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

func (h *ChannelHandler) UpdateChannel(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.UpdateChannelRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	// 转换 GroupStringIDs 到 GroupIDs
	if len(req.GroupStringIDs) > 0 {
		req.GroupIDs = make([]int64, 0, len(req.GroupStringIDs))
		for _, idStr := range req.GroupStringIDs {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				req.GroupIDs = append(req.GroupIDs, id)
			}
		}
	}
	resp, err := h.ChannelService.Update(ctx.Request.Context(), id, req)
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

func (h *ChannelHandler) SetChannelStatus(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.SetChannelStatusRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.ChannelService.UpdateStatus(ctx.Request.Context(), id, req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ChannelHandler) ReplaceChannelGroups(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.ReplaceChannelGroupsRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.ChannelService.ReplaceGroups(ctx.Request.Context(), id, req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ChannelHandler) DeleteChannel(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	if err := h.ChannelService.Delete(ctx.Request.Context(), id); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *ChannelHandler) FetchChannelRelays(ctx *gin.Context) {
	var req v1.FetchModelRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.RelayService.FetchModel(ctx.Request.Context(), &req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}
