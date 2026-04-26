package handler

import (
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/middleware"
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
	// Public routes - login endpoint (no auth)
	users := r.Group("/user")
	{
		users.POST("/login", h.Login)
	}

	// Protected routes - require JWT auth
	self := r.Group("/self")
	self.Use(middleware.RequireAuth(h.Service.JWT()))
	{
		self.GET("", h.GetUserByID)
		self.PUT("", h.UpdateUser)
		self.PATCH("/password", h.UpdateUserPassword)
	}
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

func (h *UserHandler) GetUserByID(ctx *gin.Context) {
	id := GetUserIdFromCtx(ctx)
	if id <= 0 {
		v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("user not found"), nil)
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
	id := GetUserIdFromCtx(ctx)
	if id <= 0 {
		v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("user not found"), nil)
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

func (h *UserHandler) UpdateUserPassword(ctx *gin.Context) {
	id := GetUserIdFromCtx(ctx)
	if id <= 0 {
		v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("user not found"), nil)
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
