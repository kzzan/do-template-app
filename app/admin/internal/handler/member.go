package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	userv1 "do-template-app/api/pb/userv1"
	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/pkg/grpcx"
)

// MemberHandler 终端用户管理：经 gRPC 调用 user 服务取数（本地不持有用户表）。
type MemberHandler struct {
	GrpcConn *grpcx.ClientConn `do:""`
}

func NewMemberHandler(injector do.Injector) (*MemberHandler, error) {
	return do.MustInvokeStruct[*MemberHandler](injector), nil
}

// List GET /v1/admin/members?page=&pageSize=（RBAC：api:/v1/admin/members + GET）
func (h *MemberHandler) List(ctx *gin.Context) {
	page := queryInt(ctx, "page", 1)
	pageSize := queryInt(ctx, "pageSize", 10)

	client := userv1.NewUserServiceClient(h.GrpcConn.ClientConn)
	resp, err := client.ListUsers(ctx.Request.Context(), &userv1.ListUsersRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		v1.HandleError(ctx, http.StatusBadGateway, v1.ErrBadGateway, nil)
		return
	}

	items := make([]map[string]any, 0, len(resp.GetList()))
	for _, u := range resp.GetList() {
		items = append(items, map[string]any{
			"userId": u.GetUserId(), "nickname": u.GetNickname(), "email": u.GetEmail(),
		})
	}
	v1.HandleSuccess(ctx, map[string]any{"list": items, "total": resp.GetTotal()})
}

func queryInt(ctx *gin.Context, key string, def int) int {
	if v, err := strconv.Atoi(ctx.Query(key)); err == nil {
		return v
	}
	return def
}
