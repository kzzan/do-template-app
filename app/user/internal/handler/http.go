package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	v1 "do-template-app/app/user/api/v1"
	"do-template-app/app/user/internal/service"
	"do-template-app/pkg/middleware"
)

// UserHTTP 终端用户 HTTP CRUD；依赖通过 do:"" 标签由容器自动注入。
type UserHTTP struct {
	userService service.UserService `do:""`
}

func NewUserHTTP(injector do.Injector) (*UserHTTP, error) {
	return do.MustInvokeStruct[*UserHTTP](injector), nil
}

// Register POST /v1/register
func (h *UserHTTP) Register(ctx *gin.Context) {
	req := new(v1.RegisterRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.userService.Register(ctx, req); err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// Login POST /v1/login
func (h *UserHTTP) Login(ctx *gin.Context) {
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	token, err := h.userService.Login(ctx, &req)
	if err != nil {
		v1.HandleError(ctx, http.StatusUnauthorized, err, nil)
		return
	}
	v1.HandleSuccess(ctx, v1.LoginResponseData{AccessToken: token})
}

// GetProfile GET /v1/user（Bearer 自查）
func (h *UserHTTP) GetProfile(ctx *gin.Context) {
	userID := middleware.GetUserIDFromCtx(ctx)
	user, err := h.userService.GetProfile(ctx, userID)
	if err != nil {
		v1.HandleError(ctx, http.StatusNotFound, err, nil)
		return
	}
	v1.HandleSuccess(ctx, v1.ProfileResponseData{
		UserID: user.UserID, Nickname: user.Nickname, Email: user.Email,
	})
}

// UpdateProfile PUT /v1/user（Bearer 自改）
func (h *UserHTTP) UpdateProfile(ctx *gin.Context) {
	var req v1.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.userService.UpdateProfile(ctx, middleware.GetUserIDFromCtx(ctx), &req); err != nil {
		zerolog.Ctx(ctx.Request.Context()).Error().Err(err).Msg("userService.UpdateProfile")
		v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// Delete DELETE /v1/user（Bearer 自删）
func (h *UserHTTP) Delete(ctx *gin.Context) {
	if err := h.userService.Delete(ctx, middleware.GetUserIDFromCtx(ctx)); err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}
