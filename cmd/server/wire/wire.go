//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/handler"
	"github.com/RenaLio/tudou/internal/pkg/app"
	"github.com/RenaLio/tudou/internal/pkg/jwt"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/pkg/server/http"
	"github.com/RenaLio/tudou/internal/pkg/sid"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/router"
	"github.com/RenaLio/tudou/internal/server"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/RenaLio/tudou/internal/start"
	"github.com/google/wire"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	repository.NewCache,
	repository.NewRepository,
	repository.NewTransaction,

	repository.NewAIModelRepo,
	repository.NewChannelRepo,
	repository.NewChannelGroupRepo,
	repository.NewTokenRepo,
	repository.NewUserRepo,
	repository.NewSystemConfigRepo,
)

var depsSet = wire.NewSet(
	jwt.NewJwt,
	sid.NewSid,
)

var serviceSet = wire.NewSet(
	service.NewService,
	service.NewAIModelService,
	service.NewChannelService,
	service.NewChannelGroupService,
	service.NewTokenService,
	service.NewUserService,
	service.NewSystemConfigService,
)

var handlerSet = wire.NewSet(
	handler.NewHandler,
	handler.NewModelHandler,
	handler.NewChannelHandler,
	handler.NewChannelGroupHandler,
	handler.NewTokenHandler,
	handler.NewUserHandler,
	handler.NewSystemConfigHandler,
)

var serverSet = wire.NewSet(server.NewHttpServer, server.NewMigrate)

func newApp(httpServer *http.Server) *app.App {
	return app.NewApp(
		app.WithServer(httpServer),
		app.WithName("tudou"),
	)
}

func BuildApp(*config.Config, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		depsSet,
		serviceSet,
		handlerSet,
		serverSet,
		wire.Struct(new(router.Deps), "*"),
		newApp,
	))
}

func InitApp(*config.Config, *log.Logger) error {
	panic(wire.Build(
		repositorySet,
		serverSet,
		start.InitApp,
	))
}
