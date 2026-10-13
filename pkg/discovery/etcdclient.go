package discovery

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// realClient 包装 clientv3 以匹配 clientCloser 接口。
type realClient struct {
	cli *clientv3.Client
}

func newEtcdClient(endpoints []string) (*realClient, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &realClient{cli: cli}, nil
}

// Get 返回前缀下全部注册值（实例地址）。
func (c *realClient) Get(ctx context.Context, prefix string) ([]string, error) {
	resp, err := c.cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Value))
	}
	return addrs, nil
}

// Watch 前缀变更通知（简化为事件信号，收到后重新 Get 全量）。
func (c *realClient) Watch(ctx context.Context, prefix string) <-chan struct{} {
	events := make(chan struct{})
	wch := c.cli.Watch(ctx, prefix, clientv3.WithPrefix())
	go func() {
		defer close(events)
		for range wch {
			select {
			case events <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events
}

// Close 关闭底层连接。
func (c *realClient) Close() error { return c.cli.Close() }
