// Package repository admin 服务数据访问（基座见 pkg/database）。
package repository

import "github.com/samber/do/v2"

// Package 注册 admin 服务的实体 repo（基座见 pkg/database.Package）。
var Package = do.Package(
	do.Lazy(NewAdminRepository),
)
