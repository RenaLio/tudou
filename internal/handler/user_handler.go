package handler

import (
	"errors"
	"net/http"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	*Handler
	UserService service.UserService
}

func NewUserHandler(base *Handler, userService service.UserService) *UserHandler {
	return &UserHandler{
		Handler:     base,
		UserService: userService,
	}
}

func (h *UserHandler) RegisterRoutes(r gin.IRouter) {
	users := r.Group("/users")
	users.POST("", h.CreateUser)
	users.POST("/login", h.Login)
	users.GET("", h.ListUsers)
	users.GET("/:id", h.GetUserByID)
	users.PUT("/:id", h.UpdateUser)
	users.PATCH("/:id/status", h.SetUserStatus)
	users.PATCH("/:id/password", h.UpdateUserPassword)
	users.DELETE("/:id", h.DeleteUser)
}

func (h *UserHandler) CreateUser(ctx *gin.Context) {
	var req v1.CreateUserRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.UserService.Create(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *UserHandler) Login(ctx *gin.Context) {
	var req v1.UserLoginRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.UserService.Login(ctx.Request.Context(), req, ctx.ClientIP())
	if err != nil {
		if err.Error() == "invalid username or password" {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage(err.Error()), nil)
			return
		}
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *UserHandler) ListUsers(ctx *gin.Context) {
	var req v1.ListUsersRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	resp, err := h.UserService.List(ctx.Request.Context(), req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *UserHandler) GetUserByID(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	resp, err := h.UserService.GetByID(ctx.Request.Context(), id, true)
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

func (h *UserHandler) UpdateUser(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.UpdateUserRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.UserService.Update(ctx.Request.Context(), id, req)
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

func (h *UserHandler) SetUserStatus(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.SetUserStatusRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	resp, err := h.UserService.UpdateStatus(ctx.Request.Context(), id, req)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, resp)
}

func (h *UserHandler) UpdateUserPassword(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req v1.UpdateUserPasswordRequest
	if !h.BindJSON(ctx, &req) {
		return
	}
	if err := h.UserService.UpdatePassword(ctx.Request.Context(), id, req); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, map[string]any{"id": id})
}

func (h *UserHandler) DeleteUser(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	if err := h.UserService.Delete(ctx.Request.Context(), id); err != nil {
		HandleServiceError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
