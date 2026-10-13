package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"golang.org/x/crypto/bcrypt"

	v1 "do-template-app/app/user/api/v1"
	"do-template-app/app/user/internal/repository"
	"do-template-app/model"
	"do-template-app/pkg/jwt"
)

// UserService 用户业务接口（gRPC 与 HTTP handler 共同消费）。
type UserService interface {
	Register(ctx context.Context, req *v1.RegisterRequest) error
	Login(ctx context.Context, req *v1.LoginRequest) (string, error)
	GetProfile(ctx context.Context, userID string) (*model.User, error)
	UpdateProfile(ctx context.Context, userID string, req *v1.UpdateProfileRequest) error
	Delete(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, page, pageSize int32) ([]model.User, int64, error)
	CountUsers(ctx context.Context) (int64, error)
}

func NewUserService(injector do.Injector) (UserService, error) {
	return &userService{
		userRepo: do.MustInvoke[repository.UserRepository](injector),
		jwt:      do.MustInvoke[*jwt.JWT](injector),
	}, nil
}

type userService struct {
	userRepo repository.UserRepository
	jwt      *jwt.JWT
}

// Register 邮箱查重后创建用户。
func (s *userService) Register(ctx context.Context, req *v1.RegisterRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return v1.ErrInternalServerError
	}
	if user != nil {
		return v1.ErrEmailAlreadyUse
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.Create(ctx, &model.User{
		UserID:   uuid.NewString(),
		Nickname: req.Nickname,
		Password: string(hashed),
		Email:    req.Email,
	})
}

// Login 校验密码后签发 90 天 token。
func (s *userService) Login(ctx context.Context, req *v1.LoginRequest) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return "", v1.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", v1.ErrUnauthorized
	}
	return s.jwt.GenToken(user.UserID, time.Now().Add(90*24*time.Hour))
}

// GetProfile 查无记录返回 ErrNotFound（HTTP 404）。
func (s *userService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, v1.ErrNotFound
	}
	return user, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID string, req *v1.UpdateProfileRequest) error {
	user, err := s.GetProfile(ctx, userID)
	if err != nil {
		return err
	}
	user.Nickname, user.Email = req.Nickname, req.Email
	return s.userRepo.Update(ctx, user)
}

func (s *userService) Delete(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}

func (s *userService) ListUsers(ctx context.Context, page, pageSize int32) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.userRepo.List(ctx, page, pageSize)
}

func (s *userService) CountUsers(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
}
