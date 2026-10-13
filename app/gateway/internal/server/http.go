// Package server 网关装配：按路径把请求反代到经 etcd 发现的后端服务。
package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/pkg/discovery"
	"do-template-app/pkg/log"
	pkgmw "do-template-app/pkg/middleware"
)

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewHTTPServer),
)

// HTTPServer 网关：单入口反代 admin/home；实现 ShutdownerWithContextAndError。
type HTTPServer struct {
	Config   *viper.Viper `do:""`
	Logger   *log.Logger  `do:""`
	Resolver discoveryResolver
	*http.Server
}

// discoveryResolver 限定网关所需的最小解析能力（便于测试替身）。
type discoveryResolver interface {
	ResolveOne(ctx context.Context, service string) (string, error)
	Close() error
}

func NewHTTPServer(injector do.Injector) (*HTTPServer, error) {
	s := do.MustInvokeStruct[*HTTPServer](injector)
	// 接口字段手动注入：InvokeStruct 按具体类型名匹配，无法填充接口
	s.Resolver = do.MustInvoke[*discovery.Resolver](injector)

	if s.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(pkgmw.Recovery(), pkgmw.RequestLogger(s.Logger), pkgmw.CORS())

	engine.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"service": "gateway",
			"routes": gin.H{
				"/v1/site/*": "home",
				"/v1/*":      "admin",
			},
		})
	})
	engine.GET("/healthz", func(ctx *gin.Context) {
		if _, err := s.Resolver.ResolveOne(ctx.Request.Context(), discovery.ServiceAdmin); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "etcd or admin unreachable"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	engine.Any("/v1/*path", s.proxy)

	s.Server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Config.GetString("http.host"), s.Config.GetInt("http.port")),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s, nil
}

// proxy 路由规则：/v1/site/* → home，其余 /v1/* → admin；透传不改写路径。
func (s *HTTPServer) proxy(ctx *gin.Context) {
	service := discovery.ServiceAdmin
	if strings.HasPrefix(ctx.Request.URL.Path, "/v1/site/") {
		service = discovery.ServiceHome
	}

	addr, err := s.Resolver.ResolveOne(ctx.Request.Context(), service)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "no backend for " + service})
		return
	}

	target := &url.URL{Scheme: "http", Host: addr}
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(req *http.Request) {
		original(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		s.Logger.Error().Err(err).Str("backend", addr).Msg("gateway proxy")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":502,"message":"backend unreachable"}`))
	}
	proxy.ServeHTTP(ctx.Writer, ctx.Request)
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
