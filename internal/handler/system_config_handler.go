package handler

import (
	"errors"
	"net/http"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SystemConfigHandler struct {
	*Handler
	SystemConfigService service.SystemConfigService
}

func NewSystemConfigHandler(base *Handler, configService service.SystemConfigService) *SystemConfigHandler {
	return &SystemConfigHandler{
		Handler:             base,
		SystemConfigService: configService,
	}
}

func (h *SystemConfigHandler) RegisterRoutes(r gin.IRouter) {
	configs := r.Group("/system-configs")
	configs.POST("", h.CreateSystemConfig)
	configs.PUT("", h.UpsertSystemConfig)
	configs.POST("/init-defaults", h.InitDefaultSystemConfigs)
	configs.GET("", h.ListSystemConfigs)
	configs.GET("/:id", h.GetSystemConfigByID)
	configs.GET("/key/:key", h.GetSystemConfigByKey)
	configs.PATCH("/key/:key/value", h.SetSystemConfigValueByKey)
	configs.DELETE("/key/:key", h.DeleteSystemConfigByKey)
}

func (h *SystemConfigHandler) CreateSystemConfig(ctx *gin.Context) {
	var req v1.CreateSystemConfigRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.SystemConfigService.Create(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *SystemConfigHandler) UpsertSystemConfig(ctx *gin.Context) {
	var req v1.UpsertSystemConfigRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.SystemConfigService.Upsert(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *SystemConfigHandler) InitDefaultSystemConfigs(ctx *gin.Context) {
	if err := h.SystemConfigService.InitDefaults(ctx.Request.Context()); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, map[string]any{"ok": true})
}

func (h *SystemConfigHandler) ListSystemConfigs(ctx *gin.Context) {
	var req v1.ListSystemConfigsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.SystemConfigService.List(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *SystemConfigHandler) GetSystemConfigByID(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}

	resp, err := h.SystemConfigService.GetByID(ctx.Request.Context(), id)
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

func (h *SystemConfigHandler) GetSystemConfigByKey(ctx *gin.Context) {
	key := strings.TrimSpace(ctx.Param("key"))
	if key == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("key is required"), nil)
		return
	}

	resp, err := h.SystemConfigService.GetByKey(ctx.Request.Context(), key)
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

func (h *SystemConfigHandler) SetSystemConfigValueByKey(ctx *gin.Context) {
	key := strings.TrimSpace(ctx.Param("key"))
	if key == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("key is required"), nil)
		return
	}
	var req v1.SetSystemConfigValueRequest
	if !h.BindJSON(ctx, &req) {
		return
	}

	resp, err := h.SystemConfigService.SetValueByKey(ctx.Request.Context(), key, req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *SystemConfigHandler) DeleteSystemConfigByKey(ctx *gin.Context) {
	key := strings.TrimSpace(ctx.Param("key"))
	if key == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("key is required"), nil)
		return
	}
	if err := h.SystemConfigService.DeleteByKey(ctx.Request.Context(), key); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
