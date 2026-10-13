package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/pkg/jwt"
)

// GetUserIDFromCtx 从 gin 上下文取出 JWT claims 中的用户 ID。
func GetUserIDFromCtx(ctx *gin.Context) string {
	claims, ok := ctx.Get("claims")
	if !ok {
		return ""
	}
	if c, ok := claims.(*jwt.MyCustomClaims); ok {
		return c.UserID
	}
	return ""
}

// StrictAuth 强制鉴权：Authorization 头必填，解析失败 401。
func StrictAuth(j *jwt.JWT) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.Request.Header.Get("Authorization")
		if tokenString == "" {
			zerolog.Ctx(ctx.Request.Context()).Warn().
				Str("url", ctx.Request.URL.String()).
				Msg("no token")
			v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
			ctx.Abort()
			return
		}

		claims, err := j.ParseToken(tokenString)
		if err != nil {
			zerolog.Ctx(ctx.Request.Context()).Error().
				Str("url", ctx.Request.URL.String()).
				Err(err).
				Msg("parse token")
			v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
			ctx.Abort()
			return
		}

		ctx.Set("claims", claims)
		ctx.Next()
	}
}

// NoStrictAuth 宽松鉴权：token 缺失或无效时放行（匿名），有效时注入 claims。
// 取值顺序：Header → Cookie → Query。
func NoStrictAuth(j *jwt.JWT) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.Request.Header.Get("Authorization")
		if tokenString == "" {
			tokenString, _ = ctx.Cookie("accessToken")
		}
		if tokenString == "" {
			tokenString = ctx.Query("accessToken")
		}
		if tokenString == "" {
			ctx.Next()
			return
		}

		claims, err := j.ParseToken(tokenString)
		if err != nil {
			ctx.Next()
			return
		}
		ctx.Set("claims", claims)
		ctx.Next()
	}
}
