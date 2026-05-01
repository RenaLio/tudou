//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/handler"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/pkg/app"
	"github.com/RenaLio/tudou/internal/pkg/jwt"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/pkg/server/http"
	"github.com/RenaLio/tudou/internal/pkg/sid"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/router"
	"github.com/RenaLio/tudou/internal/server"
	"github.com/RenaLio/tudou/internal/server/task"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/RenaLio/tudou/internal/start"
	"github.com/RenaLio/tudou/internal/tasks"
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
	repository.NewChannelStatsRepo,
	repository.NewChannelModelStatsRepo,
	repository.NewTokenStatsRepo,
	repository.NewUserStatsRepo,
	repository.NewUserUsageDailyStatsRepo,
	repository.NewUserUsageHourlyStatsRepo,
	repository.NewRequestLogRepo,
	repository.NewAggregationTaskRepo,
	repository.NewUserRepo,
	repository.NewSystemConfigRepo,
)

var depsSet = wire.NewSet(
	jwt.NewJwt,
	sid.NewSid,
	start.InitLBRegistry,
	wire.Bind(new(service.LBRegistryReloader), new(*loadbalancer.Registry)),
	wire.Bind(new(service.GroupRegistryReloader), new(*loadbalancer.Registry)),
	wire.Bind(new(handler.RegistryHelper), new(*loadbalancer.Registry)),
	loadbalancer.NewDynamicLoadBalancer,
	wire.Bind(new(loadbalancer.LoadBalancer), new(*loadbalancer.DynamicLoadBalancer)),
	newAsyncMetricsCollector,
	wire.Bind(new(loadbalancer.MetricsCollector), new(*loadbalancer.AsyncMetricsCollector)),
)

func newAsyncMetricsCollector(reg *loadbalancer.Registry) *loadbalancer.AsyncMetricsCollector {
	return loadbalancer.NewAsyncMetricsCollector(reg, 1024)
}

var serviceSet = wire.NewSet(
	service.NewService,
	service.NewAIModelService,
	service.NewChannelService,
	service.NewChannelGroupService,
	service.NewTokenService,
	service.NewUserService,
	service.NewSystemConfigService,
	service.NewRelayService,
	wire.Bind(new(handler.RelayService), new(*service.RelayService)),
	service.NewStatsService,
	wire.Bind(new(handler.StatsService), new(*service.StatsService)),
	service.NewRequestLogService,
	wire.Bind(new(service.RequestLogService), new(*service.RequestLogServiceImpl)),
	wire.Bind(new(service.RequestLogCreator), new(*service.RequestLogServiceImpl)),
)

var handlerSet = wire.NewSet(
	handler.NewHandler,
	handler.NewModelHandler,
	handler.NewChannelHandler,
	handler.NewChannelGroupHandler,
	handler.NewTokenHandler,
	handler.NewUserHandler,
	handler.NewSystemConfigHandler,
	handler.NewStatsHandler,
	handler.NewRelayHandler,
	handler.NewRequestLogHandler,
	handler.NewDebugHelperHandler,
	handler.NewSelectOptionHandler,
)

var serverSet = wire.NewSet(server.NewHttpServer, server.NewMigrate)

var taskSet = wire.NewSet(
	tasks.NewMockTask,
	tasks.NewStatsAggregationTask,
)

func NewTaskServer(
	logger *log.Logger,
	mockTask *tasks.MockTask,
	statsAggregationTask *tasks.StatsAggregationTask,
) *task.TaskServer {
	return task.NewTaskServer(logger, mockTask, statsAggregationTask)
}

func newApp(
	httpServer *http.Server,
	taskServer *task.TaskServer,
) *app.App {
	return app.NewApp(
		app.WithServer(httpServer, taskServer),
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
		taskSet,
		NewTaskServer,
		wire.Struct(new(router.Deps), "*"),
		newApp,
	))
}

func InitApp(*config.Config, *log.Logger) error {
	panic(wire.Build(
		repositorySet,
		depsSet,
		serviceSet,
		serverSet,
		start.InitApp,
	))
}
