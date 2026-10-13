package database

import (
	"net"
	"testing"
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/pkg/log"
)

// TestNewRedis 验证 Redis 装配（构造期 ping）；本地无 redis 时跳过（make up 可启动）。
func TestNewRedis(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:6379", 200*time.Millisecond)
	if err != nil {
		t.Skip("redis not running on 127.0.0.1:6379, run: make up")
	}
	conn.Close()

	conf := viper.New()
	conf.Set("data.redis.addr", "127.0.0.1:6379")

	injector := do.New()
	do.Provide(injector, func(do.Injector) (*viper.Viper, error) { return conf, nil })
	do.Provide(injector, func(do.Injector) (*log.Logger, error) { return log.NewLogger(viper.New()), nil })
	do.Provide(injector, NewRedis)

	rdb, err := do.Invoke[*Redis](injector)
	if err != nil {
		t.Fatalf("Invoke NewRedis: %v", err)
	}
	defer rdb.Shutdown()

	if err := rdb.HealthCheck(t.Context()); err != nil {
		t.Errorf("HealthCheck: %v", err)
	}
}
