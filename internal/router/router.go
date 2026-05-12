package router

import (
	"errors"
	"time"

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
	if deps.Conf == nil {
		return errors.New("config is nil")
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

	// Rate limiting is applied only to API/relay routes. The bundled SPA/static
	// assets must stay outside the limiter because this server is not frontend/backend separated.
	rateLimiter := buildRateLimitMiddleware(deps)

	// Public routes (no auth required)
	publicAPIGroup := engine.Group("/api/v1")
	if rateLimiter != nil {
		publicAPIGroup.Use(rateLimiter)
	}
	deps.UserHandler.RegisterRoutes(publicAPIGroup)

	// Protected routes (JWT auth required)
	apiV1Group := engine.Group("/api/v1")
	apiV1Group.Use(middleware.RequestID(deps.Logger))
	if rateLimiter != nil {
		apiV1Group.Use(rateLimiter)
	}
	apiV1Group.Use(middleware.RequireAuth(deps.UserHandler.Service.JWT()))
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
	if rateLimiter != nil {
		v1Group.Use(rateLimiter)
	}
	v1Group.Use(middleware.RequireToken(deps.TokenService))
	deps.RelayHandler.RegisterRoutes(v1Group)

	// Root-level /models route (same auth as /v1/models)
	rootGroup := engine.Group("")
	rootGroup.Use(middleware.RequestID(deps.Logger))
	if rateLimiter != nil {
		rootGroup.Use(rateLimiter)
	}
	rootGroup.Use(middleware.RequireToken(deps.TokenService))
	rootGroup.GET("/models", deps.RelayHandler.TokenModels)

	return nil
}

func buildRateLimitMiddleware(deps *Deps) gin.HandlerFunc {
	if !deps.Conf.Http.RateLimit.Enabled {
		return nil
	}
	return middleware.RateLimit(middleware.RateLimitConfig{
		Enabled:       deps.Conf.Http.RateLimit.Enabled,
		GlobalEnabled: deps.Conf.Http.RateLimit.GlobalEnabled,
		GlobalRPS:     deps.Conf.Http.RateLimit.GlobalRPS,
		GlobalBurst:   deps.Conf.Http.RateLimit.GlobalBurst,
		IPEnabled:     deps.Conf.Http.RateLimit.IPEnabled,
		IPRPS:         deps.Conf.Http.RateLimit.IPRPS,
		IPBurst:       deps.Conf.Http.RateLimit.IPBurst,
		IPTTL:         time.Duration(deps.Conf.Http.RateLimit.IPTTLMinutes) * time.Minute,
	})
}
