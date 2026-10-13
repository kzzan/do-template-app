// Package service 业务逻辑层。
package service

import "github.com/samber/do/v2"

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewSiteService),
	do.Lazy(NewAuthService),
)
