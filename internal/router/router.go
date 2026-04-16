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

	apiV1Group := engine.Group("/api/v1")
	apiV1Group.Use(middleware.RequestID(deps.Logger))
	{
		deps.ChannelHandler.RegisterRoutes(apiV1Group)
		deps.ModelHandler.RegisterRoutes(apiV1Group)
		deps.UserHandler.RegisterRoutes(apiV1Group)
		deps.ChannelGroupHandler.RegisterRoutes(apiV1Group)
		deps.SystemConfigHandler.RegisterRoutes(apiV1Group)
		deps.TokenHandler.RegisterRoutes(apiV1Group)
		deps.StatsHandler.RegisterRoutes(apiV1Group)
	}
	return nil
}
