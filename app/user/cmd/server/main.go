package main

import (
	"context"
	"flag"
	stdlog "log"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/app/user/internal/handler"
	"do-template-app/app/user/internal/repository"
	"do-template-app/app/user/internal/server"
	"do-template-app/app/user/internal/service"
	"do-template-app/pkg/bootstrap"
	"do-template-app/pkg/database"
	"do-template-app/pkg/discovery"
	"do-template-app/pkg/jwt"
	"do-template-app/pkg/log"
)

func main() {
	confPath := flag.String("conf", "config/user/local.yml", "config path, eg: -conf ./config/user/local.yml")
	flag.Parse()

	container := do.New(
		bootstrap.BasePackage(*confPath),
		database.Package,
		repository.Package,
		service.Package,
		handler.Package,
		server.Package,
		do.Lazy(func(i do.Injector) (*jwt.JWT, error) {
			return jwt.NewJWT(do.MustInvoke[*viper.Viper](i)), nil
		}),
		discovery.PackageFor[*server.GRPCServer](discovery.ServiceUser, func(s *server.GRPCServer) string { return s.Addr() }),
	)

	grpcSrv := do.MustInvoke[*server.GRPCServer](container)
	httpSrv := do.MustInvoke[*server.HTTPServer](container)
	logger := do.MustInvoke[*log.Logger](container)
	// 双协议双注册：Registrar 先由容器创建（gRPC 地址），HTTP 地址追加共用 lease
	registrar := do.MustInvoke[*discovery.Registrar](container)
	if err := registrar.Register(discovery.ServiceUserHTTP, httpSrv.Server.Addr); err != nil {
		stdlog.Fatalf("register user-http: %v", err)
	}

	go func() {
		if err := grpcSrv.Start(); err != nil {
			stdlog.Printf("grpc server exited: %v", err)
		}
	}()
	go func() {
		if err := httpSrv.Start(); err != nil {
			stdlog.Printf("http server exited: %v", err)
		}
	}()
	logger.Info().Str("grpc", grpcSrv.Addr()).Str("http", "http://"+httpSrv.Server.Addr).Msg("user server started")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, report := container.ShutdownOnSignalsWithContext(shutdownCtx, syscall.SIGINT, syscall.SIGTERM)
	if !report.Succeed {
		stdlog.Printf("shutdown report: %v", report)
	}
}
