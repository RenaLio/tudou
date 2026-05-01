package router

import (
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/handler"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/service"
	"gorm.io/gorm"
)

type Deps struct {
	Conf                *config.Config
	Logger              *log.Logger
	ModelHandler        *handler.ModelHandler
	ChannelHandler      *handler.ChannelHandler
	ChannelGroupHandler *handler.ChannelGroupHandler
	TokenHandler        *handler.TokenHandler
	UserHandler         *handler.UserHandler
	SystemConfigHandler *handler.SystemConfigHandler
	StatsHandler        *handler.StatsHandler
	RelayHandler        *handler.RelayHandler
	RequestLogHandler   *handler.RequestLogHandler
	DebugHandler        *handler.DebugHelperHandler
	SelectOptionHandler *handler.SelectOptionHandler
	TokenService        service.TokenService
	DB                  *gorm.DB
	Registry            *loadbalancer.Registry
}
