// Package v1 user 服务的 HTTP 请求/响应 DTO 与业务错误码。
package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// HandleSuccess 输出 200 成功响应。
func HandleSuccess(ctx *gin.Context, data any) {
	if data == nil {
		data = map[string]any{}
	}
	ctx.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

// HandleError 按 err 映射的业务码输出错误响应；未注册的 err 归为 500，nil 归为 unknown。
func HandleError(ctx *gin.Context, httpCode int, err error, data any) {
	if data == nil {
		data = map[string]any{}
	}
	code, message := http.StatusInternalServerError, "unknown error"
	if err != nil {
		if c, ok := errorCodeMap[err]; ok {
			code, message = c, err.Error()
		} else {
			message = err.Error()
		}
	}
	ctx.JSON(httpCode, Response{Code: code, Message: message, Data: data})
}

var errorCodeMap = map[error]int{}

// newError 注册业务错误码，必须在包初始化期调用。
func newError(code int, msg string) error {
	err := errors.New(msg)
	errorCodeMap[err] = code
	return err
}
