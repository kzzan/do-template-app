// Package middleware gin 中间件。
package middleware

import (
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"do-template-app/pkg/log"
)

// RequestLogger 为每个请求注入带 request_id 的 logger，并输出访问日志。
func RequestLogger(base *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		requestId := uuid.NewString()[:8]

		logger := base.With().Str("request_id", requestId).Logger()
		ctx.Request = ctx.Request.WithContext(logger.WithContext(ctx.Request.Context()))

		ctx.Next()

		var event *zerolog.Event
		if ctx.Writer.Status() >= 500 {
			event = logger.Error()
		} else {
			event = logger.Info()
		}
		event.
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Int("status", ctx.Writer.Status()).
			Dur("latency", time.Since(start)).
			Str("client_ip", ctx.ClientIP()).
			Msg("access")
	}
}

// Recovery panic 恢复，输出 zerolog 日志并返回 500。
func Recovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				zerolog.Ctx(ctx.Request.Context()).Error().
					Any("panic", err).
					Str("stack", string(debug.Stack())).
					Msg("recovered from panic")
				ctx.AbortWithStatus(500)
			}
		}()
		ctx.Next()
	}
}
