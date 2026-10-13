// Package server 服务器装配：gin 引擎、中间件、路由与优雅启停。
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/app/admin/internal/handler"
	"do-template-app/app/admin/internal/middleware"
	"do-template-app/app/admin/internal/repository"
	"do-template-app/pkg/database"
	"do-template-app/pkg/jwt"
	"do-template-app/pkg/log"
	pkgmw "do-template-app/pkg/middleware"
)

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewHTTPServer),
)

// HTTPServer 包装 *http.Server；依赖经 do:"" 标签由容器在构造期注入，
// 实现 ShutdownerWithContextAndError 参与容器优雅关闭。
type HTTPServer struct {
	Config        *viper.Viper               `do:""`
	Logger        *log.Logger                `do:""`
	JWT           *jwt.JWT                   `do:""`
	DB            *database.Database         `do:""`
	AdminHandler  *handler.AdminHandler      `do:""`
	MemberHandler *handler.MemberHandler     `do:""`
	AdminRepo     repository.AdminRepository `do:""`
	*http.Server
}

func NewHTTPServer(injector do.Injector) (*HTTPServer, error) {
	s := do.MustInvokeStruct[*HTTPServer](injector)

	if s.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(pkgmw.Recovery(), pkgmw.RequestLogger(s.Logger), pkgmw.CORS())

	engine.GET("/", func(ctx *gin.Context) {
		v1.HandleSuccess(ctx, gin.H{"message": "Welcome to do-template-app (admin)"})
	})
	engine.GET("/healthz", s.healthCheck)

	v1Group := engine.Group("/v1")
	v1Group.POST("/login", s.AdminHandler.Login)

	// 管理接口：登录 + RBAC 双中间件
	admin := v1Group.Group("/admin", pkgmw.StrictAuth(s.JWT), middleware.RBAC(s.AdminRepo))
	{
		admin.GET("/users", s.AdminHandler.GetAdminUsers)
		admin.POST("/users", s.AdminHandler.AdminUserCreate)
		admin.PUT("/users", s.AdminHandler.AdminUserUpdate)
		admin.DELETE("/users", s.AdminHandler.AdminUserDelete)

		admin.GET("/roles", s.AdminHandler.GetRoles)
		admin.POST("/roles", s.AdminHandler.RoleCreate)
		admin.PUT("/roles", s.AdminHandler.RoleUpdate)
		admin.DELETE("/roles", s.AdminHandler.RoleDelete)

		admin.GET("/permissions", s.AdminHandler.GetRolePermissions)
		admin.PUT("/permissions", s.AdminHandler.UpdateRolePermissions)

		// 终端用户管理：经 gRPC 调 user 服务
		admin.GET("/members", s.MemberHandler.List)
	}

	s.Server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Config.GetString("http.host"), s.Config.GetInt("http.port")),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s, nil
}

// healthCheck 直连 Database 依赖做 ping；
// 不在 struct 里保存 injector 运行时调用（官方文档反模式）。
func (s *HTTPServer) healthCheck(ctx *gin.Context) {
	checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
	defer cancel()

	if err := s.DB.HealthCheck(checkCtx); err != nil {
		v1.HandleError(ctx, http.StatusServiceUnavailable, v1.ErrInternalServerError, err)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// Start 阻塞监听；返回 http.ErrServerClosed 表示正常关闭。
func (s *HTTPServer) Start() error {
	if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 实现 do.ShutdownerWithContextAndError，容器关闭时被调用。
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
