// Package handler HTTP 处理器。
package handler

import "github.com/samber/do/v2"

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewSiteHandler),
	do.Lazy(NewAuthHandler),
)
