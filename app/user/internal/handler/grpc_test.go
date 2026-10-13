package handler

import (
	"context"
	"net"
	"testing"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	userv1 "do-template-app/api/pb/userv1"
	"do-template-app/app/user/internal/service"
	"do-template-app/model"
)

// fakeUserService 内存实现的最小替身。
type fakeUserService struct {
	service.UserService
	users []model.User
}

func (f fakeUserService) GetProfile(_ context.Context, userId string) (*model.User, error) {
	for i, u := range f.users {
		if u.UserID == userId {
			return &f.users[i], nil
		}
	}
	return nil, context.DeadlineExceeded
}

func (f fakeUserService) ListUsers(_ context.Context, page, pageSize int32) ([]model.User, int64, error) {
	start := (page - 1) * pageSize
	if start >= int32(len(f.users)) {
		return nil, int64(len(f.users)), nil
	}
	end := start + pageSize
	if end > int32(len(f.users)) {
		end = int32(len(f.users))
	}
	return f.users[start:end], int64(len(f.users)), nil
}

func (f fakeUserService) CountUsers(context.Context) (int64, error) {
	return int64(len(f.users)), nil
}

func newFakeGrpc(t *testing.T, svc service.UserService) userv1.UserServiceClient {
	t.Helper()
	injector := do.New()
	do.Provide(injector, func(do.Injector) (service.UserService, error) { return svc, nil })
	do.Provide(injector, NewUserGrpc)

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, do.MustInvoke[*UserGrpc](injector))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return userv1.NewUserServiceClient(conn)
}

func TestUserGrpc(t *testing.T) {
	client := newFakeGrpc(t, fakeUserService{users: []model.User{
		{UserID: "u-1", Nickname: "alice", Email: "a@b.com"},
		{UserID: "u-2", Nickname: "bob", Email: "b@b.com"},
	}})

	profile, err := client.GetProfile(t.Context(), &userv1.GetProfileRequest{UserId: "u-1"})
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.Nickname != "alice" || profile.Email != "a@b.com" {
		t.Errorf("profile = %+v", profile)
	}

	list, err := client.ListUsers(t.Context(), &userv1.ListUsersRequest{Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if list.Total != 2 || len(list.List) != 1 {
		t.Errorf("ListUsers = total %d len %d, want 2/1", list.Total, len(list.List))
	}

	count, err := client.CountUsers(t.Context(), &userv1.CountUsersRequest{})
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count.Total != 2 {
		t.Errorf("CountUsers = %d, want 2", count.Total)
	}
}
