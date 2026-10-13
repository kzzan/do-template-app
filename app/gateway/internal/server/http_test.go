package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeResolver 固定地址映射的解析器替身（无需 etcd）。
type fakeResolver map[string]string

func (f fakeResolver) ResolveOne(_ context.Context, service string) (string, error) {
	if addr, ok := f[service]; ok {
		return addr, nil
	}
	return "", fmt.Errorf("no instance for service %q", service)
}

func (f fakeResolver) Close() error { return nil }

func newTestGateway(t *testing.T, adminSrv, homeSrv *httptest.Server) *HTTPServer {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return &HTTPServer{Resolver: fakeResolver{
		"admin": strings.TrimPrefix(adminSrv.URL, "http://"),
		"home":  strings.TrimPrefix(homeSrv.URL, "http://"),
	}}
}

// TestProxyRouting 验证路由规则与透传：/v1/site/* → home，其余 → admin。
func TestProxyRouting(t *testing.T) {
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"backend":"admin"}`))
	}))
	defer admin.Close()
	home := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"backend":"home"}`))
	}))
	defer home.Close()

	s := newTestGateway(t, admin, home)

	// 真实 HTTP 服务承载（ReverseProxy 需要 ResponseWriter 实现 CloseNotifier，
	// ResponseRecorder 不满足）
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Any("/v1/*path", s.proxy)
	gateway := httptest.NewServer(engine)
	defer gateway.Close()

	tests := []struct {
		name     string
		path     string
		wantBody string
	}{
		{name: "site → home", path: "/v1/site/health", wantBody: `{"backend":"home"}`},
		{name: "login → admin", path: "/v1/login", wantBody: `{"backend":"admin"}`},
		{name: "admin api → admin", path: "/v1/admin/users?page=1", wantBody: `{"backend":"admin"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(gateway.URL + tt.path)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.wantBody {
				t.Errorf("body = %s, want %s", body, tt.wantBody)
			}
		})
	}
}

// TestProxyNoBackend 后端无实例时 502。
func TestProxyNoBackend(t *testing.T) {
	s := &HTTPServer{Resolver: fakeResolver{}}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/v1/login", nil)
	s.proxy(ctx)
	if w.Code != 502 {
		t.Errorf("code = %d, want 502", w.Code)
	}
}
