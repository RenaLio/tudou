package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ModelService interface {
	Create(ctx context.Context, req v1.CreateAIModelRequest) (*v1.AIModelResponse, error)
	GetByName(ctx context.Context, name string) (*v1.AIModelResponse, error)
	List(ctx context.Context, req v1.ListAIModelsRequest) (*v1.ListResponse[v1.AIModelResponse], error)
	Update(ctx context.Context, name string, req v1.UpdateAIModelRequest) (*v1.AIModelResponse, error)
	Delete(ctx context.Context, name string) error
}

var _ ModelService = (*service.AIModelService)(nil)

type ModelHandler struct {
	*Handler
	ModelService ModelService
}

func NewModelHandler(base *Handler, modelService ModelService) *ModelHandler {
	return &ModelHandler{
		Handler:      base,
		ModelService: modelService,
	}
}

func (h *ModelHandler) RegisterRoutes(r gin.IRouter) {
	models := r.Group("/model")
	models.POST("", h.CreateAIModel)
	models.GET("", h.ListAIModels)
	models.GET("/:name", h.GetAIModelByName)
	models.PUT("/:name", h.UpdateAIModel)
	models.DELETE("/:name", h.DeleteAIModel)
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

func (h *ModelHandler) GetAIModelByName(ctx *gin.Context) {
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("name is required"), nil)
		return
	}

	resp, err := h.ModelService.GetByName(ctx.Request.Context(), name)
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
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("name is required"), nil)
		return
	}

	var req v1.UpdateAIModelRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	// Model name is immutable via API updates.
	req.Name = nil

	resp, err := h.ModelService.Update(ctx.Request.Context(), name, req)
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
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("name is required"), nil)
		return
	}
	if err := h.ModelService.Delete(ctx.Request.Context(), name); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
