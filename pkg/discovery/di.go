package discovery

import (
	"github.com/samber/do/v2"
	"github.com/spf13/viper"
)

// PackageFor 返回一个把 Registrar 注册进容器的 Package：
// 地址取自服务实例（如 HTTPServer.Server.Addr），容器关闭时自动注销。
// 泛型 S 为服务类型，addrOf 从实例提取对外地址。
func PackageFor[S any](service string, addrOf func(S) string) func(do.Injector) {
	return do.Package(
		do.Lazy(func(i do.Injector) (*Registrar, error) {
			conf := do.MustInvoke[*viper.Viper](i)
			addr := addrOf(do.MustInvoke[S](i))
			return NewRegistrar(conf, service, addr)
		}),
	)
}
