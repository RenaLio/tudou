package router

import (
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/handler"
	"github.com/RenaLio/tudou/internal/pkg/log"
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
}
