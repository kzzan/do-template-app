package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"golang.org/x/crypto/bcrypt"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/app/admin/internal/repository"
	"do-template-app/model"
	"do-template-app/pkg/jwt"
)

// AdminService 管理端业务接口。
type AdminService interface {
	Login(ctx context.Context, req *v1.LoginRequest) (string, error)
	GetAdminUsers(ctx context.Context, req *v1.GetAdminUsersRequest) (*v1.GetAdminUsersResponseData, error)
	AdminUserCreate(ctx context.Context, req *v1.AdminUserCreateRequest) error
	AdminUserUpdate(ctx context.Context, req *v1.AdminUserUpdateRequest) error
	AdminUserDelete(ctx context.Context, userId string) error

	GetRoles(ctx context.Context) ([]v1.RoleItem, error)
	RoleCreate(ctx context.Context, req *v1.RoleCreateRequest) error
	RoleUpdate(ctx context.Context, req *v1.RoleUpdateRequest) error
	RoleDelete(ctx context.Context, sid string) error

	GetRolePermissions(ctx context.Context, sid string) (*v1.GetRolePermissionsResponseData, error)
	UpdateRolePermissions(ctx context.Context, req *v1.UpdateRolePermissionsRequest) error
}

func NewAdminService(injector do.Injector) (AdminService, error) {
	return &adminService{
		adminRepo: do.MustInvoke[repository.AdminRepository](injector),
		jwt:       do.MustInvoke[*jwt.JWT](injector),
	}, nil
}

type adminService struct {
	adminRepo repository.AdminRepository
	jwt       *jwt.JWT
}

func (s *adminService) Login(ctx context.Context, req *v1.LoginRequest) (string, error) {
	user, err := s.adminRepo.GetAdminUserByUsername(ctx, req.Username)
	if err != nil {
		return "", v1.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", v1.ErrUnauthorized
	}
	return s.jwt.GenToken(user.UserID, time.Now().Add(24*time.Hour))
}

func (s *adminService) GetAdminUsers(ctx context.Context, req *v1.GetAdminUsersRequest) (*v1.GetAdminUsersResponseData, error) {
	users, total, err := s.adminRepo.GetAdminUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	items := make([]v1.AdminUserItem, 0, len(users))
	for _, u := range users {
		roles, _ := s.adminRepo.GetUserRoles(ctx, u.UserID)
		if roles == nil {
			roles = []string{}
		}
		items = append(items, v1.AdminUserItem{
			UserID:    u.UserID,
			Username:  u.Username,
			Nickname:  u.Nickname,
			Email:     u.Email,
			Phone:     u.Phone,
			Roles:     roles,
			CreatedAt: u.CreatedAt.Format(time.DateTime),
			UpdatedAt: u.UpdatedAt.Format(time.DateTime),
		})
	}
	return &v1.GetAdminUsersResponseData{List: items, Total: total}, nil
}

func (s *adminService) AdminUserCreate(ctx context.Context, req *v1.AdminUserCreateRequest) error {
	if _, err := s.adminRepo.GetAdminUserByUsername(ctx, req.Username); !errors.Is(err, v1.ErrNotFound) {
		return v1.ErrUsernameAlreadyUse
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.adminRepo.AdminUserCreate(ctx, &model.AdminUser{
		UserID:   uuid.NewString(),
		Username: req.Username,
		Nickname: req.Nickname,
		Password: string(hashed),
		Email:    req.Email,
		Phone:    req.Phone,
	}, req.Roles)
}

func (s *adminService) AdminUserUpdate(ctx context.Context, req *v1.AdminUserUpdateRequest) error {
	user, err := s.adminRepo.GetAdminUser(ctx, req.UserID)
	if err != nil {
		return err
	}

	// 密码留空则不修改
	password := user.Password
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		password = string(hashed)
	}

	user.Username, user.Nickname = req.Username, req.Nickname
	user.Email, user.Phone, user.Password = req.Email, req.Phone, password
	return s.adminRepo.AdminUserUpdate(ctx, user, req.Roles)
}

func (s *adminService) AdminUserDelete(ctx context.Context, userId string) error {
	if userId == model.SuperAdminID {
		return v1.ErrSuperAdminProtected
	}
	return s.adminRepo.AdminUserDelete(ctx, userId)
}

func (s *adminService) GetRoles(ctx context.Context) ([]v1.RoleItem, error) {
	roles, err := s.adminRepo.GetRoles(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]v1.RoleItem, 0, len(roles))
	for _, r := range roles {
		items = append(items, v1.RoleItem{Sid: r.Sid, Name: r.Name})
	}
	return items, nil
}

func (s *adminService) RoleCreate(ctx context.Context, req *v1.RoleCreateRequest) error {
	return s.adminRepo.RoleCreate(ctx, model.Role{Sid: req.Sid, Name: req.Name})
}

func (s *adminService) RoleUpdate(ctx context.Context, req *v1.RoleUpdateRequest) error {
	return s.adminRepo.RoleUpdate(ctx, model.Role{Sid: req.Sid, Name: req.Name})
}

func (s *adminService) RoleDelete(ctx context.Context, sid string) error {
	return s.adminRepo.RoleDelete(ctx, sid)
}

func (s *adminService) GetRolePermissions(ctx context.Context, sid string) (*v1.GetRolePermissionsResponseData, error) {
	perms, err := s.adminRepo.GetRolePermissions(ctx, sid)
	if err != nil {
		return nil, err
	}
	items := make([]v1.Permission, 0, len(perms))
	for _, p := range perms {
		items = append(items, v1.Permission{Resource: p.Resource, Action: p.Action})
	}
	return &v1.GetRolePermissionsResponseData{Sid: sid, Permissions: items}, nil
}

func (s *adminService) UpdateRolePermissions(ctx context.Context, req *v1.UpdateRolePermissionsRequest) error {
	perms := make([]model.Permission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, model.Permission{Resource: p.Resource, Action: p.Action})
	}
	return s.adminRepo.UpdateRolePermissions(ctx, req.Sid, perms)
}
