package handler

import (
	"errors"
	"net/http"
	"strconv"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/pkg/jwt"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Logger  *log.Logger
	Service *service.Service
}

func NewHandler(logger *log.Logger, baseService *service.Service) *Handler {
	return &Handler{
		Logger:  logger,
		Service: baseService,
	}
}

func (h *Handler) Log(c *gin.Context) *log.Logger {
	return h.Logger.FromContext(c.Request.Context())
}

func (h *Handler) BindJSON(ctx *gin.Context, req any) bool {
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage(err.Error()), nil)
		return false
	}
	return true
}

func (h *Handler) BindQuery(ctx *gin.Context, req any) bool {
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage(err.Error()), nil)
		return false
	}
	return true
}

func (h *Handler) ParseIDParam(ctx *gin.Context, key string) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param(key), 10, 64)
	if err != nil || id <= 0 {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("invalid id"), nil)
		return 0, false
	}
	return id, true
}

func HandleServiceError(ctx *gin.Context, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, v1.ErrBadRequest):
		v1.Fail(ctx, v1.ErrBadRequest, nil)
	case errors.Is(err, v1.ErrUnauthorized):
		v1.Fail(ctx, v1.ErrUnauthorized, nil)
	default:
		v1.Fail(ctx, v1.ErrInternalServerError.WithMessage(err.Error()), nil)
	}
}

func HandleNotFound(ctx *gin.Context) {
	ctx.JSON(http.StatusNotFound, v1.Response{
		Code:    v1.ErrNotFound.Code,
		Message: v1.ErrNotFound.Message,
		Data:    map[string]any{},
	})
}

func GetUserIdFromCtx(ctx *gin.Context) int64 {
	if v, exists := ctx.Get(constants.UserIdKey()); exists {
		switch userID := v.(type) {
		case int64:
			if userID > 0 {
				return userID
			}
		case int:
			if userID > 0 {
				return int64(userID)
			}
		case string:
			parsed, err := strconv.ParseInt(userID, 10, 64)
			if err == nil && parsed > 0 {
				return parsed
			}
		}
	}

	v, exists := ctx.Get(constants.ClaimsKey())
	if !exists {
		return 0
	}
	claims, ok := v.(*jwt.MyCustomClaims)
	if !ok {
		return 0
	}
	userId, err := strconv.ParseInt(claims.UserId, 10, 64)
	if err != nil {
		return 0
	}
	return userId
}
