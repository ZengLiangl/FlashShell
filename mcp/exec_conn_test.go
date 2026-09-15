package mcp

import (
	"errors"
	"testing"
)

func TestSSHConnectionError(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"连接失败: dial tcp 10.8.0.1:5160: i/o timeout", true},
		{"连接错误", true},
		{"SSH 未连接", true},
		{"SSH客户端未连接", true},
		{"read tcp 10.8.0.3:57554->10.8.0.1:5160: read: connection reset by peer", true},
		{"dial tcp 10.8.0.1:5160: connect: connection refused", true},
		{"ssh: handshake failed", true},
		{"exit status 1", false},
		{"命令超时", false},
		{"[timeout] 命令超时（多数是交互式命令在等 stdin）", false},
		{"", false},
	}
	for _, c := range cases {
		var err error
		if c.msg != "" {
			err = errors.New(c.msg)
		}
		if got := sshConnectionError(err); got != c.want {
			t.Fatalf("%q: got %v want %v", c.msg, got, c.want)
		}
	}
}
