package handler

import (
	"errors"
	"net/http"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ModelHandler struct {
	*Handler
	ModelService service.AIModelService
}

func NewModelHandler(base *Handler, modelService service.AIModelService) *ModelHandler {
	return &ModelHandler{
		Handler:      base,
		ModelService: modelService,
	}
}

func (h *ModelHandler) RegisterRoutes(r gin.IRouter) {
	models := r.Group("/models")
	models.POST("", h.CreateAIModel)
	models.GET("", h.ListAIModels)
	models.GET("/:id", h.GetAIModelByID)
	models.PUT("/:id", h.UpdateAIModel)
	models.PATCH("/:id/enabled", h.SetAIModelEnabled)
	models.DELETE("/:id", h.DeleteAIModel)
}

func (h *ModelHandler) CreateAIModel(ctx *gin.Context) {
	var req v1.CreateAIModelRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.ModelService.Create(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ModelHandler) ListAIModels(ctx *gin.Context) {
	var req v1.ListAIModelsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.ModelService.List(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *ModelHandler) GetAIModelByID(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}

	resp, err := h.ModelService.GetByID(ctx.Request.Context(), id)
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

func (h *ModelHandler) UpdateAIModel(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}

	var req v1.UpdateAIModelRequest
	if !h.BindJSON(ctx, &req) {
		return
	}

	resp, err := h.ModelService.Update(ctx.Request.Context(), id, req)
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

func (h *ModelHandler) SetAIModelEnabled(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.SetAIModelEnabledRequest
	if !h.BindJSON(ctx, &req) {
		return
	}

	resp, err := h.ModelService.SetEnabled(ctx.Request.Context(), id, req)
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

func (h *ModelHandler) DeleteAIModel(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	if err := h.ModelService.Delete(ctx.Request.Context(), id); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
