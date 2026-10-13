package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/samber/do/v2"

	v1 "do-template-app/app/home/api/v1"
	"do-template-app/pkg/discovery"
)

// AuthService 经 etcd 发现 user 服务并转发登录/注册。
type AuthService interface {
	Register(ctx context.Context, req *v1.RegisterRequest) error
	Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponseData, error)
}

func NewAuthService(injector do.Injector) (AuthService, error) {
	return &authService{
		resolver: do.MustInvoke[*discovery.Resolver](injector),
		client:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

type authService struct {
	resolver *discovery.Resolver
	client   *http.Client
}

// userResp user 服务的统一响应包。
type userResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// call 按服务发现地址 POST JSON，解包 {code,message,data}。
func (s *authService) call(ctx context.Context, path string, body any, out any) error {
	addr, err := s.resolver.ResolveOne(ctx, discovery.ServiceUserHTTP)
	if err != nil {
		return fmt.Errorf("discover user-http: %w", err)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://"+addr+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("call user service: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	wrapped := userResp{Data: json.RawMessage("{}")}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return fmt.Errorf("decode user response: %w", err)
	}
	if wrapped.Code != 0 {
		return fmt.Errorf("%s", wrapped.Message)
	}
	if out != nil {
		return json.Unmarshal(wrapped.Data, out)
	}
	return nil
}

func (s *authService) Register(ctx context.Context, req *v1.RegisterRequest) error {
	return s.call(ctx, "/v1/register", req, nil)
}

func (s *authService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponseData, error) {
	data := &v1.LoginResponseData{}
	if err := s.call(ctx, "/v1/login", req, data); err != nil {
		return nil, err
	}
	return data, nil
}
