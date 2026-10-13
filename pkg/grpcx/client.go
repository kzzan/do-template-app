// Package grpcx 提供跨服务的 gRPC 客户端连接（admin/home 调 user 服务）。
package grpcx

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/spf13/viper"

	"do-template-app/pkg/discovery"
)

// ClientConn 包装 *grpc.ClientConn；实现 do.ShutdownerWithError 随容器优雅关闭。
// grpc.NewClient 懒连接，首次 RPC 才真正建立通道。
type ClientConn struct {
	*grpc.ClientConn
}

// Shutdown 实现 do.ShutdownerWithError。
func (c *ClientConn) Shutdown() error {
	return c.ClientConn.Close()
}

// NewClient 经 etcd 服务发现拨号指定服务（scheme "dtcd" 由 pkg/discovery 注册）。
// ponytail: 不加密——模板假设内网通信；跨网部署时换 TLS credentials
func NewClient(conf *viper.Viper, service string) (*ClientConn, error) {
	conn, err := grpc.NewClient(
		discovery.GRPCTarget(conf, service),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &ClientConn{ClientConn: conn}, nil
}
