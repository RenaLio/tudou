package handler

import (
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
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
	groups := r.Group("/channel-group")
	groups.POST("", h.CreateChannelGroup)
	groups.GET("", h.ListChannelGroups)
	groups.GET("/:id", h.GetChannelGroupByID)
	groups.PUT("/:id", h.UpdateChannelGroup)
	groups.DELETE("/:id", h.DeleteChannelGroup)

	// 负载均衡策略
	groups.GET("/load-balance-strategies", h.ListLoadBalanceStrategies)
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
	resp, err := h.ChannelGroupService.GetByID(ctx.Request.Context(), id)
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

// LoadBalanceStrategyInfo 负载均衡策略信息
type LoadBalanceStrategyInfo struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ListLoadBalanceStrategies 返回所有负载均衡策略
func (h *ChannelGroupHandler) ListLoadBalanceStrategies(ctx *gin.Context) {
	strategies := []LoadBalanceStrategyInfo{
		{
			Value:       string(models.LoadBalanceStrategyRandom),
			Label:       "随机",
			Description: "随机选择一个可用渠道",
		},
		{
			Value:       string(models.LoadBalanceStrategyPerformance),
			Label:       "综合性能优先",
			Description: "根据 TTFT、TPS、成功率综合评估，选择性能最优的渠道",
		},
		{
			Value:       string(models.LoadBalanceStrategyTTFTFirst),
			Label:       "响应时间优先",
			Description: "选择首字延迟(TTFT)最低的渠道",
		},
		{
			Value:       string(models.LoadBalanceStrategyTPSFirst),
			Label:       "TPS优先",
			Description: "选择每秒输出token数最高的渠道",
		},
		{
			Value:       string(models.LoadBalanceStrategySuccessFirst),
			Label:       "成功率优先",
			Description: "选择请求成功率最高的渠道",
		},
		{
			Value:       string(models.LoadBalanceStrategyCostFirst),
			Label:       "成本优先",
			Description: "选择成本最低的渠道",
		},
		{
			Value:       string(models.LoadBalanceStrategyWeighted),
			Label:       "加权",
			Description: "根据渠道权重进行加权随机选择",
		},
		{
			Value:       string(models.LoadBalanceStrategyLeastConn),
			Label:       "最少连接",
			Description: "选择当前连接数最少的渠道",
		},
	}
	v1.Success(ctx, strategies)
}
