package define

import (
	"time"

	"golang.org/x/crypto/ssh"
)

const sshKeepaliveInterval = 20 * time.Second

// BindSSHClient 接管已握手的 SSH 客户端：标记存活、Wait 掉线、周期 keepalive。
func (rm *RemoteMachine) BindSSHClient(client *ssh.Client) {
	if rm == nil {
		return
	}
	rm.stopKeepalive()
	rm.SFTPClient = nil
	rm.SSHClient = client
	if client == nil {
		rm.alive.Store(false)
		return
	}
	rm.alive.Store(true)
	stop := make(chan struct{})
	rm.kaMu.Lock()
	rm.kaStop = stop
	rm.kaMu.Unlock()
	go rm.waitSSH(client, stop)
	go rm.keepaliveSSH(client, stop)
}

func (rm *RemoteMachine) waitSSH(client *ssh.Client, stop <-chan struct{}) {
	_ = client.Wait()
	select {
	case <-stop:
		return
	default:
	}
	rm.alive.Store(false)
}

func (rm *RemoteMachine) keepaliveSSH(client *ssh.Client, stop <-chan struct{}) {
	t := time.NewTicker(sshKeepaliveInterval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				rm.alive.Store(false)
				_ = client.Close()
				return
			}
		}
	}
}

func (rm *RemoteMachine) stopKeepalive() {
	if rm == nil {
		return
	}
	rm.kaMu.Lock()
	defer rm.kaMu.Unlock()
	if rm.kaStop != nil {
		close(rm.kaStop)
		rm.kaStop = nil
	}
}
