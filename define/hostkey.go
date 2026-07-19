package define

import "golang.org/x/crypto/ssh"

var hostKeyCallback ssh.HostKeyCallback = ssh.InsecureIgnoreHostKey()

// SetHostKeyCallback 设置全局 SSH Host Key 校验回调（由 app 启动时注入）
func SetHostKeyCallback(cb ssh.HostKeyCallback) {
	if cb != nil {
		hostKeyCallback = cb
	}
}

func currentHostKeyCallback() ssh.HostKeyCallback {
	return hostKeyCallback
}
