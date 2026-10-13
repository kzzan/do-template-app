package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"

	v1 "do-template-app/app/admin/api/v1"
	"do-template-app/model"
	"do-template-app/pkg/database"
	"do-template-app/pkg/log"
)

const testSchema = `
CREATE TABLE IF NOT EXISTS admin_users (
    user_id VARCHAR(36) PRIMARY KEY, username VARCHAR(50) NOT NULL, nickname VARCHAR(50) NOT NULL DEFAULT '',
    password VARCHAR(128) NOT NULL, email VARCHAR(100) NOT NULL DEFAULT '', phone VARCHAR(20) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS roles (sid VARCHAR(50) PRIMARY KEY, name VARCHAR(100) NOT NULL, created_at TIMESTAMP NOT NULL);
CREATE TABLE IF NOT EXISTS user_roles (user_id VARCHAR(36) NOT NULL, role_sid VARCHAR(50) NOT NULL, PRIMARY KEY (user_id, role_sid));
CREATE TABLE IF NOT EXISTS role_permissions (
    role_sid VARCHAR(50) NOT NULL, resource VARCHAR(255) NOT NULL, action VARCHAR(20) NOT NULL,
    PRIMARY KEY (role_sid, resource, action)
);
`

func newTestRepo(t *testing.T) (*database.Repository, func()) {
	t.Helper()
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	r := database.New(log.NewLogger(viper.New()), &database.Database{DB: db})
	return r, func() { db.Close() }
}

func TestAdminUserCRUDAndPagination(t *testing.T) {
	r, cleanup := newTestRepo(t)
	defer cleanup()
	repo := &adminRepository{Repository: r}
	ctx := context.Background()

	for _, name := range []string{"alice", "bob", "carol"} {
		if err := repo.AdminUserCreate(ctx, &model.AdminUser{
			UserID: "u-" + name, Username: name, Password: "hash", Nickname: name,
		}, nil); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	users, total, err := repo.GetAdminUsers(ctx, &v1.GetAdminUsersRequest{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("GetAdminUsers: %v", err)
	}
	if total != 3 || len(users) != 2 {
		t.Errorf("page1: total=%d len=%d, want 3/2", total, len(users))
	}

	filtered, total, err := repo.GetAdminUsers(ctx, &v1.GetAdminUsersRequest{Page: 1, PageSize: 10, Username: "ali"})
	if err != nil {
		t.Fatalf("GetAdminUsers filter: %v", err)
	}
	if total != 1 || filtered[0].Username != "alice" {
		t.Errorf("filter: total=%d, want alice", total)
	}

	byName, err := repo.GetAdminUserByUsername(ctx, "bob")
	if err != nil || byName.UserID != "u-bob" {
		t.Errorf("GetAdminUserByUsername: %v %+v", err, byName)
	}
	if _, err := repo.GetAdminUserByUsername(ctx, "nobody"); err != v1.ErrNotFound {
		t.Errorf("missing user err = %v, want ErrNotFound", err)
	}

	if err := repo.AdminUserUpdate(ctx, &model.AdminUser{UserID: "u-bob", Username: "bob", Nickname: "bobby", Password: "h2"}, []string{"editor"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	roles, _ := repo.GetUserRoles(ctx, "u-bob")
	if len(roles) != 1 || roles[0] != "editor" {
		t.Errorf("roles after update = %v, want [editor]", roles)
	}

	if err := repo.AdminUserDelete(ctx, "u-bob"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetAdminUser(ctx, "u-bob"); err != v1.ErrNotFound {
		t.Errorf("deleted user err = %v, want ErrNotFound", err)
	}
	if roles, _ := repo.GetUserRoles(ctx, "u-bob"); len(roles) != 0 {
		t.Errorf("roles should be cleaned on delete, got %v", roles)
	}
}

// TestEnforce 验证 RBAC 判定链：user→role→(resource,action)。
func TestEnforce(t *testing.T) {
	r, cleanup := newTestRepo(t)
	defer cleanup()
	repo := &adminRepository{Repository: r}
	ctx := context.Background()

	repo.RoleCreate(ctx, model.Role{Sid: "editor", Name: "Editor"})
	repo.UpdateRolePermissions(ctx, "editor", []model.Permission{
		{Resource: "api:/v1/admin/users", Action: "GET"},
	})
	repo.AdminUserCreate(ctx, &model.AdminUser{UserID: "u-9", Username: "dave", Password: "h"}, []string{"editor"})

	tests := []struct {
		name     string
		userId   string
		resource string
		action   string
		want     bool
	}{
		{name: "allowed", userId: "u-9", resource: "api:/v1/admin/users", action: "GET", want: true},
		{name: "wrong action", userId: "u-9", resource: "api:/v1/admin/users", action: "POST", want: false},
		{name: "wrong resource", userId: "u-9", resource: "api:/v1/admin/roles", action: "GET", want: false},
		{name: "unknown user", userId: "u-x", resource: "api:/v1/admin/users", action: "GET", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Enforce(ctx, tt.userId, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("Enforce: %v", err)
			}
			if got != tt.want {
				t.Errorf("Enforce(%s,%s,%s) = %v, want %v", tt.userId, tt.resource, tt.action, got, tt.want)
			}
		})
	}
}

// TestRoleLifecycle 验证角色删除连带清理权限与用户绑定。
func TestRoleLifecycle(t *testing.T) {
	r, cleanup := newTestRepo(t)
	defer cleanup()
	repo := &adminRepository{Repository: r}
	ctx := context.Background()

	repo.RoleCreate(ctx, model.Role{Sid: "tmp", Name: "Temp"})
	repo.UpdateRolePermissions(ctx, "tmp", []model.Permission{{Resource: "api:/x", Action: "GET"}})
	repo.AdminUserCreate(ctx, &model.AdminUser{UserID: "u-1", Username: "eve", Password: "h"}, []string{"tmp"})

	if err := repo.RoleDelete(ctx, "tmp"); err != nil {
		t.Fatalf("RoleDelete: %v", err)
	}
	if roles, _ := repo.GetRoles(ctx); len(roles) != 0 {
		t.Errorf("roles should be empty, got %v", roles)
	}
	if got, err := repo.Enforce(ctx, "u-1", "api:/x", "GET"); err != nil || got {
		t.Errorf("Enforce after role delete = %v (err %v), want false", got, err)
	}
}
