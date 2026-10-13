// Package handler HTTP 处理器。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	v1 "do-template-app/app/home/api/v1"
	"do-template-app/app/home/internal/service"
)

// SiteHandler 依赖通过 do:"" 标签由容器自动注入。
type SiteHandler struct {
	siteService service.SiteService `do:""`
}

func NewSiteHandler(injector do.Injector) (*SiteHandler, error) {
	return do.MustInvokeStruct[*SiteHandler](injector), nil
}

// Health GET /v1/site/health
func (h *SiteHandler) Health(ctx *gin.Context) {
	v1.HandleSuccess(ctx, h.siteService.Health())
}

// Meta GET /v1/site/meta
func (h *SiteHandler) Meta(ctx *gin.Context) {
	v1.HandleSuccess(ctx, h.siteService.Meta())
}

// Manifest GET /v1/site/manifest
func (h *SiteHandler) Manifest(ctx *gin.Context) {
	v1.HandleSuccess(ctx, h.siteService.Manifest())
}

// Stats GET /v1/site/stats：站点统计（用户数来自 user 服务）
func (h *SiteHandler) Stats(ctx *gin.Context) {
	stats, err := h.siteService.Stats(ctx.Request.Context())
	if err != nil {
		v1.HandleError(ctx, http.StatusBadGateway, err)
		return
	}
	v1.HandleSuccess(ctx, stats)
}
