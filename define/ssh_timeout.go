package define

import (
	"sync/atomic"
	"time"
)

const defaultSSHHandshakeTimeout = 30 * time.Second

var sshHandshakeTimeoutNs atomic.Int64

func init() {
	sshHandshakeTimeoutNs.Store(int64(defaultSSHHandshakeTimeout))
}

// SetSSHHandshakeTimeout 设置全局 SSH 握手超时（TCP + SSH 协商）
func SetSSHHandshakeTimeout(d time.Duration) {
	if d <= 0 {
		d = defaultSSHHandshakeTimeout
	}
	sshHandshakeTimeoutNs.Store(int64(d))
}

// SSHHandshakeTimeout 当前 SSH 握手超时
func SSHHandshakeTimeout() time.Duration {
	if ns := sshHandshakeTimeoutNs.Load(); ns > 0 {
		return time.Duration(ns)
	}
	return defaultSSHHandshakeTimeout
}
