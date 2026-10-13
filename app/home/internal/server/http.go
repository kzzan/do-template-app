// Package server home 服务器装配：gin 引擎、中间件与优雅启停（无 DB）。
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/app/home/internal/handler"
	"do-template-app/pkg/log"
	pkgmw "do-template-app/pkg/middleware"
)

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewHTTPServer),
)

// HTTPServer 包装 *http.Server；实现 ShutdownerWithContextAndError 参与容器优雅关闭。
type HTTPServer struct {
	Config      *viper.Viper         `do:""`
	Logger      *log.Logger          `do:""`
	SiteHandler *handler.SiteHandler `do:""`
	AuthHandler *handler.AuthHandler `do:""`
	*http.Server
}

func NewHTTPServer(injector do.Injector) (*HTTPServer, error) {
	s := do.MustInvokeStruct[*HTTPServer](injector)

	if s.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(pkgmw.Recovery(), pkgmw.RequestLogger(s.Logger), pkgmw.CORS())

	auth := engine.Group("/v1/auth")
	{
		auth.POST("/register", s.AuthHandler.Register)
		auth.POST("/login", s.AuthHandler.Login)
	}

	site := engine.Group("/v1/site")
	{
		site.GET("/health", s.SiteHandler.Health)
		site.GET("/meta", s.SiteHandler.Meta)
		site.GET("/manifest", s.SiteHandler.Manifest)
		site.GET("/stats", s.SiteHandler.Stats)
	}

	s.Server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Config.GetString("http.host"), s.Config.GetInt("http.port")),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s, nil
}

// Start 阻塞监听。
func (s *HTTPServer) Start() error {
	if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 实现 do.ShutdownerWithContextAndError。
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
