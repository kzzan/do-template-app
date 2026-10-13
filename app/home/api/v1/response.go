package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response home 应用独立的统一响应结构（与 admin 解耦，服务自治）。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func HandleSuccess(ctx *gin.Context, data any) {
	if data == nil {
		data = map[string]any{}
	}
	ctx.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

func HandleError(ctx *gin.Context, httpCode int, err error) {
	ctx.JSON(httpCode, Response{Code: httpCode, Message: err.Error(), Data: map[string]any{}})
}
