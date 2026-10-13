package main

import (
	"context"
	"flag"
	stdlog "log"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/app/gateway/internal/server"
	"do-template-app/pkg/bootstrap"
	"do-template-app/pkg/discovery"
	"do-template-app/pkg/log"
)

func main() {
	confPath := flag.String("conf", "config/gateway/local.yml", "config path, eg: -conf ./config/gateway/local.yml")
	flag.Parse()

	container := do.New(
		bootstrap.BasePackage(*confPath),
		do.Lazy(func(i do.Injector) (*discovery.Resolver, error) {
			return discovery.NewResolver(do.MustInvoke[*viper.Viper](i))
		}),
		server.Package,
	)

	httpSrv := do.MustInvoke[*server.HTTPServer](container)
	logger := do.MustInvoke[*log.Logger](container)

	go func() {
		if err := httpSrv.Start(); err != nil {
			stdlog.Printf("gateway exited: %v", err)
		}
	}()
	logger.Info().Str("addr", "http://"+httpSrv.Server.Addr).Msg("gateway started")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, report := container.ShutdownOnSignalsWithContext(shutdownCtx, syscall.SIGINT, syscall.SIGTERM)
	if !report.Succeed {
		stdlog.Printf("shutdown report: %v", report)
	}
}
