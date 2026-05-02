package server

import (
	"errors"
	"fmt"
	nethttp "net/http"
	"strings"

	"github.com/RenaLio/tudou/internal/middleware"
	"github.com/RenaLio/tudou/internal/pkg/server/http"
	"github.com/RenaLio/tudou/internal/router"
	"github.com/RenaLio/tudou/web"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewHttpServer(deps *router.Deps) *http.Server {
	if deps == nil {
		panic("router deps is nil")
	}
	if deps.Conf == nil {
		panic("router deps conf is nil")
	}
	if deps.Logger == nil {
		panic("router deps logger is nil")
	}

	if deps.Conf.Env == "prod" && !deps.Conf.Debug.Default {
		gin.SetMode(gin.ReleaseMode)
	}

	host := deps.Conf.Http.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := deps.Conf.Http.Port
	if port <= 0 {
		port = 8080
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	s := http.NewServer(
		engine,
		deps.Logger,
		http.WithServerHost(host),
		http.WithServerPort(port),
	)

	fileSystem, err := static.EmbedFolder(web.Assets(), "dist")
	if err != nil {
		panic(err)
	}
	s.Use(static.Serve("/", fileSystem))
	s.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") || strings.HasPrefix(c.Request.URL.Path, "/v1") {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		c.FileFromFS("dist/index.html", nethttp.FS(web.Assets()))
	})

	s.Use(gin.Recovery())

	if err := router.RegisterHTTPRoutes(s.Engine, deps); err != nil {
		deps.Logger.Error("register http routes failed", zap.Error(err), zap.String("addr", fmt.Sprintf("%s:%d", host, port)))
		panic(errors.New("register http routes failed"))
	}

	s.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{
			"status": "ok",
		})
	})

	return s
}
