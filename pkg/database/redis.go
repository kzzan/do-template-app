package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-app/pkg/log"
)

// Redis 包装 *redis.Client；实现 ShutdownerWithError 与 HealthcheckerWithContext，
// 懒加载——仅有业务真正 MustInvoke 时才建立连接。
type Redis struct {
	*redis.Client
}

// Shutdown 实现 do.ShutdownerWithError。
func (r *Redis) Shutdown() error {
	return r.Client.Close()
}

// HealthCheck 实现 do.HealthcheckerWithContext。
func (r *Redis) HealthCheck(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// NewRedis 按配置建立连接并 ping 验证（5s 超时）。
func NewRedis(injector do.Injector) (*Redis, error) {
	conf := do.MustInvoke[*viper.Viper](injector)

	rdb := redis.NewClient(&redis.Options{
		Addr:         conf.GetString("data.redis.addr"),
		Password:     conf.GetString("data.redis.password"),
		DB:           conf.GetInt("data.redis.db"),
		ReadTimeout:  conf.GetDuration("data.redis.read_timeout"),
		WriteTimeout: conf.GetDuration("data.redis.write_timeout"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	do.MustInvoke[*log.Logger](injector).Info().Str("addr", conf.GetString("data.redis.addr")).Msg("redis connected")
	return &Redis{Client: rdb}, nil
}
