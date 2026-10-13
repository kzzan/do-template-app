package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	v1 "do-template-app/app/user/api/v1"
	"do-template-app/app/user/internal/handler"
	"do-template-app/pkg/database"
	"do-template-app/pkg/jwt"
	"do-template-app/pkg/log"
	pkgmw "do-template-app/pkg/middleware"
)

// HTTPServer user 服务的 HTTP 入口；实现 ShutdownerWithContextAndError。
type HTTPServer struct {
	Config   *viper.Viper       `do:""`
	Logger   *log.Logger        `do:""`
	DB       *database.Database `do:""`
	JWT      *jwt.JWT           `do:""`
	UserHTTP *handler.UserHTTP  `do:""`
	*http.Server
}

func NewHTTPServer(injector do.Injector) (*HTTPServer, error) {
	s := do.MustInvokeStruct[*HTTPServer](injector)

	if s.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(pkgmw.Recovery(), pkgmw.RequestLogger(s.Logger), pkgmw.CORS())

	engine.GET("/healthz", s.healthCheck)

	v1Group := engine.Group("/v1")
	v1Group.POST("/register", s.UserHTTP.Register)
	v1Group.POST("/login", s.UserHTTP.Login)

	authed := v1Group.Group("/", pkgmw.StrictAuth(s.JWT))
	{
		authed.GET("/user", s.UserHTTP.GetProfile)
		authed.PUT("/user", s.UserHTTP.UpdateProfile)
		authed.DELETE("/user", s.UserHTTP.Delete)
	}

	s.Server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Config.GetString("http.host"), s.Config.GetInt("http.port")),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s, nil
}

func (s *HTTPServer) healthCheck(ctx *gin.Context) {
	checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
	defer cancel()
	if err := s.DB.HealthCheck(checkCtx); err != nil {
		v1.HandleError(ctx, http.StatusServiceUnavailable, v1.ErrInternalServerError, err)
		return
	}
	v1.HandleSuccess(ctx, nil)
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
