package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/app/admin/internal/repository"
	"do-template-app/model"
	pkgmw "do-template-app/pkg/middleware"
)

// RBAC 权限检查：须在 StrictAuth 之后使用（依赖 claims）。
// 超管（user_id = "1"）跳过检查；其余按 user→role→(resource, action) 判定。
func RBAC(adminRepo repository.AdminRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := pkgmw.GetUserIDFromCtx(ctx)
		if userID == "" {
			v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
			ctx.Abort()
			return
		}
		if userID == model.SuperAdminID {
			ctx.Next()
			return
		}

		resource := model.ResourcePrefixAPI + ctx.Request.URL.Path
		action := ctx.Request.Method
		allowed, err := adminRepo.Enforce(ctx.Request.Context(), userID, resource, action)
		if err != nil || !allowed {
			v1.HandleError(ctx, http.StatusForbidden, v1.ErrForbidden, nil)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
