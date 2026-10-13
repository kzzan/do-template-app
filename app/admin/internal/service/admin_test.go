package service

import (
	"context"
	"testing"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/app/admin/internal/repository"
	"do-template-app/model"
	"do-template-app/pkg/jwt"
)

// fakeAdminRepo 内存实现的管理员仓库替身。
type fakeAdminRepo struct {
	users map[string]*model.AdminUser // key: username
}

func (f *fakeAdminRepo) GetAdminUsers(context.Context, *v1.GetAdminUsersRequest) ([]model.AdminUser, int64, error) {
	users := make([]model.AdminUser, 0, len(f.users))
	for _, u := range f.users {
		users = append(users, *u)
	}
	return users, int64(len(users)), nil
}

func (f *fakeAdminRepo) GetAdminUser(_ context.Context, userId string) (*model.AdminUser, error) {
	for _, u := range f.users {
		if u.UserID == userId {
			return u, nil
		}
	}
	return nil, v1.ErrNotFound
}

func (f *fakeAdminRepo) GetAdminUserByUsername(_ context.Context, username string) (*model.AdminUser, error) {
	if u, ok := f.users[username]; ok {
		return u, nil
	}
	return nil, v1.ErrNotFound
}

func (f *fakeAdminRepo) AdminUserCreate(_ context.Context, user *model.AdminUser, _ []string) error {
	f.users[user.Username] = user
	return nil
}

func (f *fakeAdminRepo) AdminUserUpdate(_ context.Context, user *model.AdminUser, _ []string) error {
	f.users[user.Username] = user
	return nil
}

func (f *fakeAdminRepo) AdminUserDelete(_ context.Context, userId string) error {
	for name, u := range f.users {
		if u.UserID == userId {
			delete(f.users, name)
			return nil
		}
	}
	return v1.ErrNotFound
}

func (f *fakeAdminRepo) GetRoles(context.Context) ([]model.Role, error) { return nil, nil }
func (f *fakeAdminRepo) RoleCreate(context.Context, model.Role) error   { return nil }
func (f *fakeAdminRepo) RoleUpdate(context.Context, model.Role) error   { return nil }
func (f *fakeAdminRepo) RoleDelete(context.Context, string) error       { return nil }

func (f *fakeAdminRepo) GetUserRoles(context.Context, string) ([]string, error) {
	return []string{}, nil
}
func (f *fakeAdminRepo) GetRolePermissions(_ context.Context, _ string) ([]model.Permission, error) {
	return nil, nil
}
func (f *fakeAdminRepo) UpdateRolePermissions(context.Context, string, []model.Permission) error {
	return nil
}
func (f *fakeAdminRepo) Enforce(context.Context, string, string, string) (bool, error) {
	return false, nil
}

// newTestService 用 do 容器装配被测服务（官方推荐的测试方式）。
func newTestService(repo repository.AdminRepository) AdminService {
	injector := do.New()
	do.Provide(injector, func(do.Injector) (repository.AdminRepository, error) { return repo, nil })
	do.Provide(injector, func(do.Injector) (*jwt.JWT, error) { return jwt.NewJWT(viper.New()), nil })
	do.Provide(injector, NewAdminService)
	return do.MustInvoke[AdminService](injector)
}

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return string(h)
}

func TestAdminLogin(t *testing.T) {
	svc := newTestService(&fakeAdminRepo{users: map[string]*model.AdminUser{
		"admin": {UserID: "1", Username: "admin", Password: hashedPassword(t, "admin123")},
	}})
	ctx := context.Background()

	token, err := svc.Login(ctx, &v1.LoginRequest{Username: "admin", Password: "admin123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	tests := []struct {
		name string
		req  v1.LoginRequest
	}{
		{name: "wrong password", req: v1.LoginRequest{Username: "admin", Password: "bad"}},
		{name: "unknown user", req: v1.LoginRequest{Username: "ghost", Password: "admin123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Login(ctx, &tt.req); err != v1.ErrUnauthorized {
				t.Errorf("Login err = %v, want ErrUnauthorized", err)
			}
		})
	}
}

func TestAdminUserCreateDuplicateUsername(t *testing.T) {
	svc := newTestService(&fakeAdminRepo{users: map[string]*model.AdminUser{
		"admin": {UserID: "1", Username: "admin", Password: "h"},
	}})

	req := &v1.AdminUserCreateRequest{Username: "admin", Password: "123456"}
	if err := svc.AdminUserCreate(context.Background(), req); err != v1.ErrUsernameAlreadyUse {
		t.Fatalf("duplicate create err = %v, want ErrUsernameAlreadyUse", err)
	}
}

// TestSuperAdminProtected 超管不可删除。
func TestSuperAdminProtected(t *testing.T) {
	svc := newTestService(&fakeAdminRepo{users: map[string]*model.AdminUser{
		"admin": {UserID: "1", Username: "admin", Password: "h"},
	}})

	if err := svc.AdminUserDelete(context.Background(), "1"); err != v1.ErrSuperAdminProtected {
		t.Fatalf("delete super admin err = %v, want ErrSuperAdminProtected", err)
	}
}
