// Package handler HTTP 处理器：参数校验 + 响应组装，不含业务逻辑。
package handler

import (
	"github.com/samber/do/v2"
)

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewAdminHandler),
	do.Lazy(NewMemberHandler),
)
