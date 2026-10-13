package main

import (
	"context"
	"flag"
	stdlog "log"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/app/home/internal/handler"
	"do-template-app/app/home/internal/server"
	"do-template-app/app/home/internal/service"
	"do-template-app/pkg/bootstrap"
	"do-template-app/pkg/discovery"
	"do-template-app/pkg/grpcx"
	"do-template-app/pkg/log"
)

func main() {
	confPath := flag.String("conf", "config/home/local.yml", "config path, eg: -conf ./config/home/local.yml")
	flag.Parse()

	container := do.New(
		bootstrap.BasePackage(*confPath),
		do.Lazy(func(i do.Injector) (*grpcx.ClientConn, error) {
			return grpcx.NewClient(do.MustInvoke[*viper.Viper](i), discovery.ServiceUser)
		}),
		do.Lazy(func(i do.Injector) (*discovery.Resolver, error) {
			return discovery.NewResolver(do.MustInvoke[*viper.Viper](i))
		}),
		service.Package,
		handler.Package,
		server.Package,
		discovery.PackageFor[*server.HTTPServer](discovery.ServiceHome, func(s *server.HTTPServer) string { return s.Server.Addr }),
	)

	httpSrv := do.MustInvoke[*server.HTTPServer](container)
	logger := do.MustInvoke[*log.Logger](container)
	do.MustInvoke[*discovery.Registrar](container)

	go func() {
		if err := httpSrv.Start(); err != nil {
			stdlog.Printf("http server exited: %v", err)
		}
	}()
	logger.Info().Str("addr", "http://"+httpSrv.Server.Addr).Msg("home server started")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, report := container.ShutdownOnSignalsWithContext(shutdownCtx, syscall.SIGINT, syscall.SIGTERM)
	if !report.Succeed {
		stdlog.Printf("shutdown report: %v", report)
	}
}
