// Package model 管理端数据模型（db 标签对应表列）。
package model

import "time"

// SuperAdminID 超管固定 user_id；RBAC 中间件对其跳过权限检查（防呆设计，对齐 nunu）。
const SuperAdminID = "1"

const (
	ResourcePrefixAPI = "api:"
)

// AdminUser 管理员账号；user_id 由应用侧生成（超管由 seed 固定为 "1"）。
type AdminUser struct {
	UserID    string    `db:"user_id"`
	Username  string    `db:"username"`
	Nickname  string    `db:"nickname"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	Phone     string    `db:"phone"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Role 角色：sid 即角色标识（如 "editor"）。
type Role struct {
	Sid      string    `db:"sid"`
	Name     string    `db:"name"`
	CreateAt time.Time `db:"created_at"`
}

// UserRole 用户-角色绑定。
type UserRole struct {
	UserID  string `db:"user_id"`
	RoleSid string `db:"role_sid"`
}

// Permission 权限点：resource 形如 "api:/v1/admin/users"，action 为 HTTP 方法。
type Permission struct {
	Resource string `db:"resource"`
	Action   string `db:"action"`
}
