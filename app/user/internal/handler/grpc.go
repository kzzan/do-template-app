// Package handler gRPC 服务实现。
package handler

import (
	"context"
	"github.com/samber/do/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userv1 "do-template-app/api/pb/userv1"
	"do-template-app/app/user/internal/service"
)

// UserGrpc 实现 userv1.UserServiceServer；
// 内嵌 Unimplemented 保证 proto 未实现的方法返回 codes.Unimplemented。
type UserGrpc struct {
	userv1.UnimplementedUserServiceServer
	userService service.UserService `do:""`
}

func NewUserGrpc(injector do.Injector) (*UserGrpc, error) {
	return do.MustInvokeStruct[*UserGrpc](injector), nil
}

func (g *UserGrpc) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.Profile, error) {
	user, err := g.userService.GetProfile(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &userv1.Profile{UserId: user.UserID, Nickname: user.Nickname, Email: user.Email}, nil
}

func (g *UserGrpc) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	users, total, err := g.userService.ListUsers(ctx, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &userv1.ListUsersResponse{Total: total, List: []*userv1.UserItem{}}
	for _, u := range users {
		resp.List = append(resp.List, &userv1.UserItem{UserId: u.UserID, Nickname: u.Nickname, Email: u.Email})
	}
	return resp, nil
}

func (g *UserGrpc) CountUsers(ctx context.Context, _ *userv1.CountUsersRequest) (*userv1.CountUsersResponse, error) {
	total, err := g.userService.CountUsers(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.CountUsersResponse{Total: total}, nil
}
