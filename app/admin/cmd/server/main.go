package main

import (
	"context"
	"flag"
	stdlog "log"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/app/admin/internal/handler"
	"do-template-app/app/admin/internal/repository"
	"do-template-app/app/admin/internal/server"
	"do-template-app/app/admin/internal/service"
	"do-template-app/pkg/bootstrap"
	"do-template-app/pkg/database"
	"do-template-app/pkg/discovery"
	"do-template-app/pkg/grpcx"
	"do-template-app/pkg/jwt"
	"do-template-app/pkg/log"
)

func main() {
	confPath := flag.String("conf", "config/admin/local.yml", "config path, eg: -conf ./config/admin/local.yml")
	flag.Parse()

	container := do.New(
		bootstrap.BasePackage(*confPath),
		do.Lazy(func(i do.Injector) (*jwt.JWT, error) {
			return jwt.NewJWT(do.MustInvoke[*viper.Viper](i)), nil
		}),
		do.Lazy(func(i do.Injector) (*grpcx.ClientConn, error) {
			return grpcx.NewClient(do.MustInvoke[*viper.Viper](i), discovery.ServiceUser)
		}),
		database.Package,
		repository.Package,
		service.Package,
		handler.Package,
		server.Package,
		discovery.PackageFor[*server.HTTPServer](discovery.ServiceAdmin, func(s *server.HTTPServer) string { return s.Server.Addr }),
	)

	httpSrv := do.MustInvoke[*server.HTTPServer](container)
	logger := do.MustInvoke[*log.Logger](container)
	// 显式触发注册（懒加载容器：不 Invoke 不实例化）
	do.MustInvoke[*discovery.Registrar](container)

	go func() {
		if err := httpSrv.Start(); err != nil {
			stdlog.Printf("http server exited: %v", err)
		}
	}()
	logger.Info().Str("addr", "http://"+httpSrv.Server.Addr).Msg("admin server started")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, report := container.ShutdownOnSignalsWithContext(shutdownCtx, syscall.SIGINT, syscall.SIGTERM)
	if !report.Succeed {
		stdlog.Printf("shutdown report: %v", report)
	}
}
