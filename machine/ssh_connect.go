package machine

import (
	"fmt"

	"FlashDock/define"
)

// ConnectRemote 建立 SSH（可选经跳板链）；withSFTP 为 true 时初始化 SFTP。
func ConnectRemote(rm *define.RemoteMachine, machine *define.Machine, withSFTP bool) error {
	if rm == nil || machine == nil {
		return fmt.Errorf("连接参数无效")
	}
	sensitive, err := machine.GetSensitiveData()
	if err != nil {
		return fmt.Errorf("获取敏感数据失败: %w", err)
	}
	targetPort := sensitive.Port
	if targetPort <= 0 {
		targetPort = 22
	}
	targetAddr := fmt.Sprintf("%s:%d", sensitive.Host, targetPort)

	auth, err := buildSSHAuth(machine, sensitive)
	if err != nil {
		return err
	}
	handshakeTimeout := define.SSHHandshakeTimeout()
	targetConfig := buildSSHClientConfig(sensitive.User, auth, machine, handshakeTimeout)

	hops, err := ResolveJumpChain(machine)
	if err != nil {
		return err
	}

	client, err := connectThroughHops(hops, targetAddr, targetConfig, machine, handshakeTimeout)
	if err != nil {
		if len(hops) == 0 {
			return fmt.Errorf("SSH连接失败: %w", err)
		}
		return err
	}

	rm.SSHClient = client
	if withSFTP {
		if err := rm.EnsureSFTP(); err != nil {
			_ = rm.Close()
			rm.SSHClient = nil
			return err
		}
	}
	return nil
}
