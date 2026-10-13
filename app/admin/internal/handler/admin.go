package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/app/admin/internal/service"
)

// AdminHandler 依赖通过 do:"" 标签由容器自动注入。
type AdminHandler struct {
	adminService service.AdminService `do:""`
}

func NewAdminHandler(injector do.Injector) (*AdminHandler, error) {
	return do.MustInvokeStruct[*AdminHandler](injector), nil
}

// Login 管理员登录
func (h *AdminHandler) Login(ctx *gin.Context) {
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	token, err := h.adminService.Login(ctx, &req)
	if err != nil {
		v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		return
	}
	v1.HandleSuccess(ctx, v1.LoginResponseData{AccessToken: token})
}

// GetAdminUsers 管理员分页列表
func (h *AdminHandler) GetAdminUsers(ctx *gin.Context) {
	var req v1.GetAdminUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	data, err := h.adminService.GetAdminUsers(ctx, &req)
	if err != nil {
		zerolog.Ctx(ctx.Request.Context()).Error().Err(err).Msg("adminService.GetAdminUsers")
		v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		return
	}
	v1.HandleSuccess(ctx, data)
}

// AdminUserCreate 创建管理员
func (h *AdminHandler) AdminUserCreate(ctx *gin.Context) {
	var req v1.AdminUserCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.adminService.AdminUserCreate(ctx, &req); err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// AdminUserUpdate 更新管理员（密码留空不改）
func (h *AdminHandler) AdminUserUpdate(ctx *gin.Context) {
	var req v1.AdminUserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.adminService.AdminUserUpdate(ctx, &req); err != nil {
		zerolog.Ctx(ctx.Request.Context()).Error().Err(err).Msg("adminService.AdminUserUpdate")
		v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// AdminUserDelete 删除管理员（超管受保护）
func (h *AdminHandler) AdminUserDelete(ctx *gin.Context) {
	var req v1.AdminUserDeleteRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.adminService.AdminUserDelete(ctx, req.UserID); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// GetRoles 角色列表
func (h *AdminHandler) GetRoles(ctx *gin.Context) {
	roles, err := h.adminService.GetRoles(ctx)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		return
	}
	v1.HandleSuccess(ctx, roles)
}

// RoleCreate 创建角色
func (h *AdminHandler) RoleCreate(ctx *gin.Context) {
	var req v1.RoleCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.adminService.RoleCreate(ctx, &req); err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// RoleUpdate 更新角色
func (h *AdminHandler) RoleUpdate(ctx *gin.Context) {
	var req v1.RoleUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.adminService.RoleUpdate(ctx, &req); err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// RoleDelete 删除角色（连带清理权限与用户绑定）
func (h *AdminHandler) RoleDelete(ctx *gin.Context) {
	var req v1.RoleDeleteRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.adminService.RoleDelete(ctx, req.Sid); err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// GetRolePermissions 查询角色权限
func (h *AdminHandler) GetRolePermissions(ctx *gin.Context) {
	data, err := h.adminService.GetRolePermissions(ctx, ctx.Query("sid"))
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, data)
}

// UpdateRolePermissions 全量替换角色权限
func (h *AdminHandler) UpdateRolePermissions(ctx *gin.Context) {
	var req v1.UpdateRolePermissionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.adminService.UpdateRolePermissions(ctx, &req); err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}
