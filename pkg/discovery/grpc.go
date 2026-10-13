package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc/resolver"
)

// etcd gRPC resolver：scheme "dtcd"，target 形如
// dtcd:///do-template/user?endpoints=127.0.0.1:2379
// 初始 Get 全量 + Watch 前缀增量，实例变化实时推送 ClientConn（换端口/扩缩容免重启）。
const scheme = "dtcd"

func init() {
	resolver.Register(etcdBuilder{})
}

type etcdBuilder struct{}

func (etcdBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	endpoints := target.URL.Query().Get("endpoints")
	if endpoints == "" {
		return nil, fmt.Errorf("dtcd target missing endpoints: %s", target.String())
	}
	prefix := strings.TrimPrefix(target.URL.Path, "/")
	if prefix == "" {
		return nil, fmt.Errorf("dtcd target missing prefix: %s", target.String())
	}

	r := &etcdResolver{
		endpoints: strings.Split(endpoints, ","),
		prefix:    prefix,
		cc:        cc,
	}
	if err := r.connect(); err != nil {
		return nil, err
	}
	go r.watch()
	return r, nil
}

func (etcdBuilder) Scheme() string { return scheme }

type etcdResolver struct {
	endpoints []string
	prefix    string
	cc        resolver.ClientConn

	mu     sync.Mutex
	cli    clientCloser
	cancel context.CancelFunc
}

// clientCloser 抽象 etcd 客户端（便于测试替换）。
type clientCloser interface {
	Get(ctx context.Context, prefix string) ([]string, error)
	Watch(ctx context.Context, prefix string) <-chan struct{}
	Close() error
}

func (r *etcdResolver) connect() error {
	c, err := newEtcdClient(r.endpoints)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.cli = c
	r.mu.Unlock()
	return nil
}

// ResolveNow 刷新当前地址集（gRPC 在连接失败时回调）。
func (r *etcdResolver) ResolveNow(resolver.ResolveNowOptions) {
	r.mu.Lock()
	cli := r.cli
	r.mu.Unlock()
	if cli == nil {
		return
	}
	addrs, err := cli.Get(context.Background(), r.prefix)
	if err != nil {
		return
	}
	_ = r.cc.UpdateState(toState(addrs))
}

func (r *etcdResolver) Close() {
	r.mu.Lock()
	cancel := r.cancel
	cli := r.cli
	r.cli = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if cli != nil {
		_ = cli.Close()
	}
}

func (r *etcdResolver) watch() {
	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	old := r.cancel
	r.cancel = cancel
	cli := r.cli
	r.mu.Unlock()
	if old != nil {
		old()
	}
	if cli == nil {
		return
	}

	addrs, err := cli.Get(ctx, r.prefix)
	if err == nil {
		_ = r.cc.UpdateState(toState(addrs))
	}

	for range cli.Watch(ctx, r.prefix) {
		addrs, err := cli.Get(ctx, r.prefix)
		if err != nil {
			return
		}
		_ = r.cc.UpdateState(toState(addrs))
	}
}

func toState(addrs []string) resolver.State {
	state := resolver.State{Addresses: make([]resolver.Address, 0, len(addrs))}
	for _, a := range addrs {
		state.Addresses = append(state.Addresses, resolver.Address{Addr: a})
	}
	return state
}
