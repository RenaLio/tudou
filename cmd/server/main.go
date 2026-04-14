package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/RenaLio/tudou/cmd/server/wire"
	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"go.uber.org/zap"
)

func main() {
	var envConf = flag.String("conf", "config/config.yaml", "config path, eg: -conf ./config/local.yml")
	flag.Parse()
	conf, err := config.NewConfig(*envConf)
	if err != nil {
		panic(err)
	}
	logger := log.NewLog(conf)

	err = wire.InitApp(conf, logger)
	if err != nil {
		panic(err)
	}

	app, cleanup, err := wire.BuildApp(conf, logger)
	defer cleanup()
	if err != nil {
		panic(err)
	}
	logger.Info("server start", zap.String("host", fmt.Sprintf("http://%s:%d", conf.Http.Host, conf.Http.Port)))
	//logger.Info("docs addr", slog.String("addr", fmt.Sprintf("http://%s:%d/swagger/index.html", conf.Http.Host, conf.Http.Port)))
	if err = app.Run(context.Background()); err != nil {
		panic(err)
	}

}
