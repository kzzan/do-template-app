// Package bootstrap 各服务 main 的公共装配基础（配置 + 日志），
// 服务特有依赖（jwt/grpcx/resolver）由各 main 追加。
package bootstrap

import (
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/pkg/config"
	"do-template-app/pkg/log"
)

// BasePackage 返回 config + log 的基础 Package。
func BasePackage(confPath string) func(do.Injector) {
	return do.Package(
		do.Lazy(func(i do.Injector) (*viper.Viper, error) {
			return config.NewConfig(confPath), nil
		}),
		do.Lazy(func(i do.Injector) (*log.Logger, error) {
			return log.NewLogger(do.MustInvoke[*viper.Viper](i)), nil
		}),
	)
}
