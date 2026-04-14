package handler

import (
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChannelGroupHandler struct {
	*Handler
	ChannelGroupService service.ChannelGroupService
}

func NewChannelGroupHandler(base *Handler, groupService service.ChannelGroupService) *ChannelGroupHandler {
	return &ChannelGroupHandler{
		Handler:             base,
		ChannelGroupService: groupService,
	}
}

func (h *ChannelGroupHandler) RegisterRoutes(r gin.IRouter) {
	groups := r.Group("/channel-groups")
	groups.POST("", h.CreateChannelGroup)
	groups.GET("", h.ListChannelGroups)
	groups.GET("/:id", h.GetChannelGroupByID)
	groups.PUT("/:id", h.UpdateChannelGroup)
	groups.PUT("/:id/channels", h.ReplaceGroupChannels)
	groups.DELETE("/:id", h.DeleteChannelGroup)
}

func (h *ChannelGroupHandler) CreateChannelGroup(ctx *gin.Context) {
	var req v1.CreateChannelGroupRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.ChannelGroupService.Create(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ChannelGroupHandler) ListChannelGroups(ctx *gin.Context) {
	var req v1.ListChannelGroupsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.ChannelGroupService.List(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ChannelGroupHandler) GetChannelGroupByID(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	resp, err := h.ChannelGroupService.GetByID(ctx.Request.Context(), id, true)
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

func (h *ChannelGroupHandler) UpdateChannelGroup(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.UpdateChannelGroupRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.ChannelGroupService.Update(ctx.Request.Context(), id, req)
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

func (h *ChannelGroupHandler) ReplaceGroupChannels(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.ReplaceGroupChannelsRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.ChannelGroupService.ReplaceChannels(ctx.Request.Context(), id, req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ChannelGroupHandler) DeleteChannelGroup(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	if err := h.ChannelGroupService.Delete(ctx.Request.Context(), id); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, nil)
}
