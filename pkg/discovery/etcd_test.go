package discovery

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// etcdReachable 本地 etcd 探测；不可达则跳过（make up 可启动）。
func etcdReachable(t *testing.T) bool {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "127.0.0.1:2379", 500*time.Millisecond)
	if err != nil {
		t.Skip("etcd not running on 127.0.0.1:2379, run: make up")
	}
	conn.Close()
	return true
}

func testConf() *viper.Viper {
	conf := viper.New()
	conf.Set("etcd.endpoints", []string{"127.0.0.1:2379"})
	return conf
}

// TestRegistrarAndResolver 注册→解析→注销 全链路。
func TestRegistrarAndResolver(t *testing.T) {
	if !etcdReachable(t) {
		return
	}
	conf := testConf()
	ctx := context.Background()

	reg, err := NewRegistrar(conf, "test-svc", "10.0.0.1:9999")
	if err != nil {
		t.Fatalf("NewRegistrar: %v", err)
	}

	resolver, err := NewResolver(conf)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	defer resolver.Close()

	// 注册键可能先于解析可见，容忍短暂不一致
	var addrs []string
	for range 10 {
		addrs, err = resolver.Resolve(ctx, "test-svc")
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil || len(addrs) != 1 || addrs[0] != "10.0.0.1:9999" {
		t.Fatalf("Resolve = %v (err %v), want [10.0.0.1:9999]", addrs, err)
	}

	if err := reg.Shutdown(); err != nil {
		t.Fatalf("registrar shutdown: %v", err)
	}

	// 注销后（缓存过期）应解析失败
	time.Sleep(cacheTTL + time.Second)
	if _, err := resolver.Resolve(ctx, "test-svc"); err == nil {
		t.Error("expected resolve error after deregister")
	}
}

// TestGRPCTarget 目标串格式。
func TestGRPCTarget(t *testing.T) {
	got := GRPCTarget(testConf(), ServiceUser)
	want := "dtcd:///do-template/user?endpoints=127.0.0.1:2379"
	if got != want {
		t.Errorf("GRPCTarget = %q, want %q", got, want)
	}
}
