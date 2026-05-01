package router

import (
	"errors"

	"github.com/RenaLio/tudou/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterHTTPRoutes(engine *gin.Engine, deps *Deps) error {
	if engine == nil {
		return errors.New("gin engine is nil")
	}
	if deps == nil {
		return errors.New("router deps is nil")
	}
	if deps.ModelHandler == nil {
		return errors.New("model handler is nil")
	}
	if deps.UserHandler == nil {
		return errors.New("user handler is nil")
	}
	if deps.UserHandler.Service == nil {
		return errors.New("base service is nil")
	}
	if deps.ChannelHandler == nil {
		return errors.New("channel handler is nil")
	}
	if deps.ChannelGroupHandler == nil {
		return errors.New("channel group handler is nil")
	}
	if deps.SystemConfigHandler == nil {
		return errors.New("system config handler is nil")
	}
	if deps.TokenHandler == nil {
		return errors.New("token handler is nil")
	}
	if deps.StatsHandler == nil {
		return errors.New("stats handler is nil")
	}
	if deps.RelayHandler == nil {
		return errors.New("relay handler is nil")
	}
	if deps.RequestLogHandler == nil {
		return errors.New("request log handler is nil")
	}

	// Public routes (no auth required)
	deps.UserHandler.RegisterRoutes(engine.Group("/api/v1"))

	// Protected routes (JWT auth required)
	apiV1Group := engine.Group("/api/v1")
	apiV1Group.Use(middleware.RequestID(deps.Logger))
	//apiV1Group.Use(middleware.RequireAuth(deps.UserHandler.Service.JWT()))
	{
		deps.ChannelHandler.RegisterRoutes(apiV1Group)
		deps.ModelHandler.RegisterRoutes(apiV1Group)
		deps.ChannelGroupHandler.RegisterRoutes(apiV1Group)
		deps.SystemConfigHandler.RegisterRoutes(apiV1Group)
		deps.TokenHandler.RegisterRoutes(apiV1Group)
		deps.StatsHandler.RegisterRoutes(apiV1Group)
		deps.RequestLogHandler.RegisterRoutes(apiV1Group)
		deps.DebugHandler.RegisterRoutes(apiV1Group)
		deps.SelectOptionHandler.RegisterRoutes(apiV1Group)
	}

	// Relay routes (Token auth required)
	v1Group := engine.Group("/v1")
	v1Group.Use(middleware.RequestID(deps.Logger))
	v1Group.Use(middleware.RequireToken(deps.TokenService))
	deps.RelayHandler.RegisterRoutes(v1Group)

	return nil
}
