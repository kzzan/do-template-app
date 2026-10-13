package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	v1 "do-template-app/app/home/api/v1"
	"do-template-app/app/home/internal/service"
)

// AuthHandler 门户登录/注册（转发 user 服务）；依赖通过 do:"" 标签由容器自动注入。
type AuthHandler struct {
	authService service.AuthService `do:""`
}

func NewAuthHandler(injector do.Injector) (*AuthHandler, error) {
	return do.MustInvokeStruct[*AuthHandler](injector), nil
}

// Register POST /v1/auth/register
func (h *AuthHandler) Register(ctx *gin.Context) {
	req := new(v1.RegisterRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := h.authService.Register(ctx.Request.Context(), req); err != nil {
		zerolog.Ctx(ctx.Request.Context()).Warn().Err(err).Msg("authService.Register")
		v1.HandleError(ctx, http.StatusBadGateway, err)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// Login POST /v1/auth/login
func (h *AuthHandler) Login(ctx *gin.Context) {
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, err)
		return
	}
	data, err := h.authService.Login(ctx.Request.Context(), &req)
	if err != nil {
		v1.HandleError(ctx, http.StatusUnauthorized, err)
		return
	}
	v1.HandleSuccess(ctx, data)
}
