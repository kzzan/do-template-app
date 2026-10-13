// Package config 加载 yaml 配置文件，支持 APP_CONF 环境变量覆盖路径。
package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// NewConfig 按路径加载配置，文件缺失或解析失败直接 panic（启动期错误不恢复）。
func NewConfig(path string) *viper.Viper {
	envConf := os.Getenv("APP_CONF")
	if envConf == "" {
		envConf = path
	}
	fmt.Println("load conf file:", envConf)

	conf := viper.New()
	conf.SetConfigFile(envConf)
	if err := conf.ReadInConfig(); err != nil {
		panic(err)
	}
	return conf
}
