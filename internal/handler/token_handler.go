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

type TokenHandler struct {
	*Handler
	TokenService service.TokenService
}

func NewTokenHandler(base *Handler, tokenService service.TokenService) *TokenHandler {
	return &TokenHandler{
		Handler:      base,
		TokenService: tokenService,
	}
}

func (h *TokenHandler) RegisterRoutes(r gin.IRouter) {
	tokens := r.Group("/token")
	tokens.POST("", h.CreateToken)
	tokens.GET("", h.ListTokens)
	tokens.GET("/:id", h.GetTokenByID)
	tokens.GET("/by-token/:token", h.GetTokenByToken)
	tokens.PUT("/:id", h.UpdateToken)
	tokens.PATCH("/:id/status", h.SetTokenStatus)
	tokens.DELETE("/:id", h.DeleteToken)
}

func (h *TokenHandler) CreateToken(ctx *gin.Context) {
	var req v1.CreateTokenRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	userId := GetUserIdFromCtx(ctx)
	if userId <= 0 {
		v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("user not found"), nil)
	}
	req.UserID = userId
	resp, err := h.TokenService.Create(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *TokenHandler) ListTokens(ctx *gin.Context) {
	var req v1.ListTokensRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.TokenService.List(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *TokenHandler) GetTokenByID(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	resp, err := h.TokenService.GetByID(ctx.Request.Context(), id, true)
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

func (h *TokenHandler) GetTokenByToken(ctx *gin.Context) {
	tokenValue := strings.TrimSpace(ctx.Param("token"))
	if tokenValue == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("token is required"), nil)
		return
	}
	resp, err := h.TokenService.GetByToken(ctx.Request.Context(), tokenValue, true)
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

func (h *TokenHandler) UpdateToken(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.UpdateTokenRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.TokenService.Update(ctx.Request.Context(), id, req)
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

func (h *TokenHandler) SetTokenStatus(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.SetTokenStatusRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.TokenService.UpdateStatus(ctx.Request.Context(), id, req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *TokenHandler) DeleteToken(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	if err := h.TokenService.Delete(ctx.Request.Context(), id); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
