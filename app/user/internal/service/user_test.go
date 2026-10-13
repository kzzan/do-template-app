package service

import (
	"context"
	"sync"
	"testing"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"

	v1 "do-template-app/app/user/api/v1"
	"do-template-app/app/user/internal/repository"
	"do-template-app/model"
	"do-template-app/pkg/jwt"
)

// fakeUserRepo 内存实现的最小替身（GetByEmail 语义：未找到返回 nil,nil）。
type fakeUserRepo struct {
	mu    sync.Mutex
	users map[string]*model.User // key: email
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return nil, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, _ string) (*model.User, error) { return nil, nil }
func (f *fakeUserRepo) List(context.Context, int32, int32) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeUserRepo) Count(context.Context) (int64, error) { return 0, nil }
func (f *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[u.Email] = u
	return nil
}
func (f *fakeUserRepo) Update(_ context.Context, u *model.User) error { return nil }
func (f *fakeUserRepo) Delete(_ context.Context, _ string) error      { return nil }

func newTestService(repo repository.UserRepository) UserService {
	injector := do.New()
	do.Provide(injector, func(do.Injector) (repository.UserRepository, error) { return repo, nil })
	do.Provide(injector, func(do.Injector) (*jwt.JWT, error) { return jwt.NewJWT(viper.New()), nil })
	do.Provide(injector, NewUserService)
	return do.MustInvoke[UserService](injector)
}

func hashed(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return string(h)
}

func TestRegisterAndLogin(t *testing.T) {
	svc := newTestService(&fakeUserRepo{users: map[string]*model.User{}})
	ctx := context.Background()

	if err := svc.Register(ctx, &v1.RegisterRequest{Email: "a@b.com", Password: "123456", Nickname: "alan"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	token, err := svc.Login(ctx, &v1.LoginRequest{Email: "a@b.com", Password: "123456"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	svc := newTestService(&fakeUserRepo{users: map[string]*model.User{}})
	ctx := context.Background()

	req := &v1.RegisterRequest{Email: "a@b.com", Password: "123456"}
	if err := svc.Register(ctx, req); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := svc.Register(ctx, req); err != v1.ErrEmailAlreadyUse {
		t.Fatalf("duplicate err = %v, want ErrEmailAlreadyUse", err)
	}
}

func TestLoginWrong(t *testing.T) {
	svc := newTestService(&fakeUserRepo{users: map[string]*model.User{
		"a@b.com": {UserID: "u-1", Email: "a@b.com", Password: hashed(t, "123456")},
	}})
	ctx := context.Background()

	tests := []struct {
		name string
		req  v1.LoginRequest
	}{
		{name: "wrong password", req: v1.LoginRequest{Email: "a@b.com", Password: "bad"}},
		{name: "unknown email", req: v1.LoginRequest{Email: "nobody@b.com", Password: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Login(ctx, &tt.req); err != v1.ErrUnauthorized {
				t.Errorf("Login err = %v, want ErrUnauthorized", err)
			}
		})
	}
}
