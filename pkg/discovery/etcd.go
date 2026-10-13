// Package discovery 提供 etcd 服务发现：注册（lease 保活/注销）与解析（HTTP 查询 + gRPC watch resolver）。
package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// KeyPrefix 所有服务注册键的统一前缀，键布局：{prefix}/{service}/{addr} → addr。
const KeyPrefix = "do-template"

// ServiceName 各服务注册到 etcd 的服务名。
const (
	ServiceAdmin    = "admin"
	ServiceHome     = "home"
	ServiceUser     = "user"
	ServiceUserHTTP = "user-http"
	ServiceGateway  = "gateway"
)

// Endpoints 从配置读取 etcd 地址列表。
func Endpoints(conf *viper.Viper) []string {
	eps := conf.GetStringSlice("etcd.endpoints")
	if len(eps) == 0 {
		eps = []string{"127.0.0.1:2379"}
	}
	return eps
}

// NewClient 建立 etcd 客户端（调用方负责 Close）。
func NewClient(conf *viper.Viper) (*clientv3.Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   Endpoints(conf),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd connect: %w", err)
	}
	return cli, nil
}

// Registrar 服务注册器：lease 保活防脑裂，Shutdown 注销（实现 do.ShutdownerWithError）。
// 同一实例可注册多个键（多协议/多地址共用一条 lease）。
type Registrar struct {
	cli     *clientv3.Client
	session *concurrency.Session
}

// NewRegistrar 以 lease（TTL 15s）注册 service/addr，进程崩溃后键自动过期消失。
func NewRegistrar(conf *viper.Viper, service, addr string) (*Registrar, error) {
	r := &Registrar{}
	if err := r.connect(conf); err != nil {
		return nil, err
	}
	if err := r.Register(service, addr); err != nil {
		r.Shutdown()
		return nil, err
	}
	return r, nil
}

func (r *Registrar) connect(conf *viper.Viper) error {
	cli, err := NewClient(conf)
	if err != nil {
		return err
	}
	session, err := concurrency.NewSession(cli, concurrency.WithTTL(15))
	if err != nil {
		cli.Close()
		return fmt.Errorf("etcd session: %w", err)
	}
	r.cli, r.session = cli, session
	return nil
}

// Register 追加注册一个 service/addr 键（复用既有 lease）。
func (r *Registrar) Register(service, addr string) error {
	key := fmt.Sprintf("%s/%s/%s", KeyPrefix, service, addr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := r.cli.Put(ctx, key, addr, clientv3.WithLease(r.session.Lease())); err != nil {
		return fmt.Errorf("etcd put %s: %w", key, err)
	}
	return nil
}

// Shutdown 实现 do.ShutdownerWithError：注销键并断开 etcd。
func (r *Registrar) Shutdown() error {
	if err := r.session.Close(); err != nil {
		return err
	}
	return r.cli.Close()
}

// Resolver HTTP 侧服务解析：按前缀取实例地址，短缓存降低 etcd 压力。
type Resolver struct {
	cli   *clientv3.Client
	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	addrs     []string
	expiresAt time.Time
}

// NewResolver 创建解析器（调用方负责 Close）。
func NewResolver(conf *viper.Viper) (*Resolver, error) {
	cli, err := NewClient(conf)
	if err != nil {
		return nil, err
	}
	return &Resolver{cli: cli, cache: map[string]cacheEntry{}}, nil
}

// Close 释放 etcd 连接。
func (r *Resolver) Close() error { return r.cli.Close() }

// cacheTTL 地址缓存有效期；略高于 lease TTL 的一半，权衡时效与 etcd 压力。
const cacheTTL = 5 * time.Second

// Resolve 返回 service 的全部实例地址（无实例返回错误）。
func (r *Resolver) Resolve(ctx context.Context, service string) ([]string, error) {
	r.mu.Lock()
	if e, ok := r.cache[service]; ok && time.Now().Before(e.expiresAt) {
		addrs := e.addrs
		r.mu.Unlock()
		return addrs, nil
	}
	r.mu.Unlock()

	prefix := KeyPrefix + "/" + service + "/"
	resp, err := r.cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd get %s: %w", prefix, err)
	}
	addrs := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Value))
	}

	r.mu.Lock()
	r.cache[service] = cacheEntry{addrs: addrs, expiresAt: time.Now().Add(cacheTTL)}
	r.mu.Unlock()
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no instance for service %q", service)
	}
	return addrs, nil
}

// ResolveOne 解析单个实例（取第一个，模板不做负载均衡）。
func (r *Resolver) ResolveOne(ctx context.Context, service string) (string, error) {
	addrs, err := r.Resolve(ctx, service)
	if err != nil {
		return "", err
	}
	return addrs[0], nil
}

// GRPCTarget 构造 gRPC 自定义 scheme 目标（pkg/discovery/grpc.go 注册 "dtcd"）。
// 形如 dtcd:///do-template/user?endpoints=127.0.0.1:2379
func GRPCTarget(conf *viper.Viper, service string) string {
	return fmt.Sprintf("dtcd:///%s/%s?endpoints=%s", KeyPrefix, service, strings.Join(Endpoints(conf), ","))
}
