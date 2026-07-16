package define

import (
	"context"
	"net"
	"sync/atomic"
	"time"
)

type dialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

var dialContextFn atomic.Value // dialContextFunc

func init() {
	dialContextFn.Store(dialContextFunc(defaultDialContext))
}

func defaultDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d := net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	return d.DialContext(ctx, network, address)
}

// SetDialContext 设置全局 TCP 拨号（用于 HTTP/SOCKS 代理）。传 nil 恢复直连。
func SetDialContext(fn dialContextFunc) {
	if fn == nil {
		dialContextFn.Store(dialContextFunc(defaultDialContext))
		return
	}
	dialContextFn.Store(fn)
}

// DialContext 使用当前拨号器建立 TCP 连接
func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	fn := dialContextFn.Load().(dialContextFunc)
	return fn(ctx, network, address)
}
