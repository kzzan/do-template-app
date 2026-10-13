package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/samber/do/v2"
	"time"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/model"
	"do-template-app/pkg/database"
)

const adminUserColumns = "user_id, username, nickname, password, email, phone, created_at, updated_at"

// AdminRepository 管理员/角色/权限数据访问接口。
type AdminRepository interface {
	GetAdminUsers(ctx context.Context, req *v1.GetAdminUsersRequest) ([]model.AdminUser, int64, error)
	GetAdminUser(ctx context.Context, userId string) (*model.AdminUser, error)
	GetAdminUserByUsername(ctx context.Context, username string) (*model.AdminUser, error)
	AdminUserCreate(ctx context.Context, user *model.AdminUser, roles []string) error
	AdminUserUpdate(ctx context.Context, user *model.AdminUser, roles []string) error
	AdminUserDelete(ctx context.Context, userId string) error

	GetRoles(ctx context.Context) ([]model.Role, error)
	RoleCreate(ctx context.Context, role model.Role) error
	RoleUpdate(ctx context.Context, role model.Role) error
	RoleDelete(ctx context.Context, sid string) error

	GetUserRoles(ctx context.Context, userId string) ([]string, error)
	GetRolePermissions(ctx context.Context, sid string) ([]model.Permission, error)
	UpdateRolePermissions(ctx context.Context, sid string, perms []model.Permission) error

	// Enforce RBAC 判定：user 经角色持有 resource+action 即放行
	Enforce(ctx context.Context, userId, resource, action string) (bool, error)
}

func NewAdminRepository(injector do.Injector) (AdminRepository, error) {
	return &adminRepository{
		Repository: do.MustInvoke[*database.Repository](injector),
	}, nil
}

type adminRepository struct {
	*database.Repository
}

var _ = database.Repository{}

func (r *adminRepository) GetAdminUsers(ctx context.Context, req *v1.GetAdminUsersRequest) ([]model.AdminUser, int64, error) {
	where, args := " WHERE 1=1", []any{}
	if req.Username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+req.Username+"%")
	}

	var total int64
	if err := r.DB(ctx).GetContext(ctx, &total, r.DB(ctx).Rebind("SELECT COUNT(*) FROM admin_users"+where), args...); err != nil {
		return nil, 0, err
	}

	query := r.DB(ctx).Rebind("SELECT " + adminUserColumns + " FROM admin_users" + where +
		" ORDER BY created_at DESC LIMIT ? OFFSET ?")
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	users := []model.AdminUser{}
	if err := r.DB(ctx).SelectContext(ctx, &users, query, args...); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *adminRepository) GetAdminUser(ctx context.Context, userId string) (*model.AdminUser, error) {
	return r.getAdminUserBy(ctx, "user_id = ?", userId)
}

func (r *adminRepository) GetAdminUserByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	return r.getAdminUserBy(ctx, "username = ?", username)
}

func (r *adminRepository) getAdminUserBy(ctx context.Context, where string, arg any) (*model.AdminUser, error) {
	var user model.AdminUser
	query := r.DB(ctx).Rebind("SELECT " + adminUserColumns + " FROM admin_users WHERE " + where)
	err := r.DB(ctx).GetContext(ctx, &user, query, arg)
	if errors.Is(err, database.ErrRecordNotFound) {
		return nil, v1.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *adminRepository) AdminUserCreate(ctx context.Context, user *model.AdminUser, roles []string) error {
	now := time.Now().UTC()
	user.CreatedAt, user.UpdatedAt = now, now

	return r.Transaction(ctx, func(ctx context.Context) error {
		query := r.DB(ctx).Rebind("INSERT INTO admin_users (" + adminUserColumns + ") VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
		if _, err := r.DB(ctx).ExecContext(ctx, query,
			user.UserID, user.Username, user.Nickname, user.Password, user.Email, user.Phone, user.CreatedAt, user.UpdatedAt,
		); err != nil {
			return err
		}
		return r.replaceUserRoles(ctx, user.UserID, roles)
	})
}

func (r *adminRepository) AdminUserUpdate(ctx context.Context, user *model.AdminUser, roles []string) error {
	user.UpdatedAt = time.Now().UTC()

	return r.Transaction(ctx, func(ctx context.Context) error {
		query := r.DB(ctx).Rebind("UPDATE admin_users SET username = ?, nickname = ?, password = ?, email = ?, phone = ?, updated_at = ? WHERE user_id = ?")
		if _, err := r.DB(ctx).ExecContext(ctx, query,
			user.Username, user.Nickname, user.Password, user.Email, user.Phone, user.UpdatedAt, user.UserID,
		); err != nil {
			return err
		}
		return r.replaceUserRoles(ctx, user.UserID, roles)
	})
}

func (r *adminRepository) AdminUserDelete(ctx context.Context, userId string) error {
	return r.Transaction(ctx, func(ctx context.Context) error {
		delUser := r.DB(ctx).Rebind("DELETE FROM admin_users WHERE user_id = ?")
		if _, err := r.DB(ctx).ExecContext(ctx, delUser, userId); err != nil {
			return err
		}
		delRoles := r.DB(ctx).Rebind("DELETE FROM user_roles WHERE user_id = ?")
		_, err := r.DB(ctx).ExecContext(ctx, delRoles, userId)
		return err
	})
}

func (r *adminRepository) GetRoles(ctx context.Context) ([]model.Role, error) {
	roles := []model.Role{}
	query := r.DB(ctx).Rebind("SELECT sid, name, created_at FROM roles ORDER BY created_at")
	err := r.DB(ctx).SelectContext(ctx, &roles, query)
	return roles, err
}

func (r *adminRepository) RoleCreate(ctx context.Context, role model.Role) error {
	role.CreateAt = time.Now().UTC()
	query := r.DB(ctx).Rebind("INSERT INTO roles (sid, name, created_at) VALUES (?, ?, ?)")
	_, err := r.DB(ctx).ExecContext(ctx, query, role.Sid, role.Name, role.CreateAt)
	return err
}

func (r *adminRepository) RoleUpdate(ctx context.Context, role model.Role) error {
	query := r.DB(ctx).Rebind("UPDATE roles SET name = ? WHERE sid = ?")
	_, err := r.DB(ctx).ExecContext(ctx, query, role.Name, role.Sid)
	return err
}

func (r *adminRepository) RoleDelete(ctx context.Context, sid string) error {
	return r.Transaction(ctx, func(ctx context.Context) error {
		delPerms := r.DB(ctx).Rebind("DELETE FROM role_permissions WHERE role_sid = ?")
		if _, err := r.DB(ctx).ExecContext(ctx, delPerms, sid); err != nil {
			return err
		}
		delBindings := r.DB(ctx).Rebind("DELETE FROM user_roles WHERE role_sid = ?")
		if _, err := r.DB(ctx).ExecContext(ctx, delBindings, sid); err != nil {
			return err
		}
		delRole := r.DB(ctx).Rebind("DELETE FROM roles WHERE sid = ?")
		_, err := r.DB(ctx).ExecContext(ctx, delRole, sid)
		return err
	})
}

func (r *adminRepository) GetUserRoles(ctx context.Context, userId string) ([]string, error) {
	roles := []string{}
	query := r.DB(ctx).Rebind("SELECT role_sid FROM user_roles WHERE user_id = ?")
	err := r.DB(ctx).SelectContext(ctx, &roles, query, userId)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *adminRepository) GetRolePermissions(ctx context.Context, sid string) ([]model.Permission, error) {
	perms := []model.Permission{}
	query := r.DB(ctx).Rebind("SELECT resource, action FROM role_permissions WHERE role_sid = ?")
	err := r.DB(ctx).SelectContext(ctx, &perms, query, sid)
	return perms, err
}

func (r *adminRepository) UpdateRolePermissions(ctx context.Context, sid string, perms []model.Permission) error {
	return r.Transaction(ctx, func(ctx context.Context) error {
		del := r.DB(ctx).Rebind("DELETE FROM role_permissions WHERE role_sid = ?")
		if _, err := r.DB(ctx).ExecContext(ctx, del, sid); err != nil {
			return err
		}
		ins := r.DB(ctx).Rebind("INSERT INTO role_permissions (role_sid, resource, action) VALUES (?, ?, ?)")
		for _, p := range perms {
			if _, err := r.DB(ctx).ExecContext(ctx, ins, sid, p.Resource, p.Action); err != nil {
				return fmt.Errorf("insert permission %s %s: %w", p.Resource, p.Action, err)
			}
		}
		return nil
	})
}

func (r *adminRepository) Enforce(ctx context.Context, userId, resource, action string) (bool, error) {
	var allowed bool
	query := r.DB(ctx).Rebind(`SELECT EXISTS(
		SELECT 1 FROM user_roles ur
		JOIN role_permissions rp ON ur.role_sid = rp.role_sid
		WHERE ur.user_id = ? AND rp.resource = ? AND rp.action = ?)`)
	err := r.DB(ctx).GetContext(ctx, &allowed, query, userId, resource, action)
	return allowed, err
}

func (r *adminRepository) replaceUserRoles(ctx context.Context, userId string, roles []string) error {
	del := r.DB(ctx).Rebind("DELETE FROM user_roles WHERE user_id = ?")
	if _, err := r.DB(ctx).ExecContext(ctx, del, userId); err != nil {
		return err
	}
	ins := r.DB(ctx).Rebind("INSERT INTO user_roles (user_id, role_sid) VALUES (?, ?)")
	for _, sid := range roles {
		if _, err := r.DB(ctx).ExecContext(ctx, ins, userId, sid); err != nil {
			return fmt.Errorf("bind role %s: %w", sid, err)
		}
	}
	return nil
}
