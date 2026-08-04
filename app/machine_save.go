package app

import (
	"strings"

	"FlashDock/define"
)

func prepareMachineForSave(machine *define.Machine, existing *define.Machine) error {
	if machine == nil {
		return nil
	}
	if machine.ProxyOverride != nil {
		define.NormalizeMachineProxyOverride(machine.ProxyOverride)
		if machine.ProxyOverride.Password == "" && existing != nil && existing.ProxyOverride != nil {
			machine.ProxyOverride.EncryptedPassword = existing.ProxyOverride.EncryptedPassword
		}
		if err := machine.ProxyOverride.PrepareMachineProxyPasswordForSave(); err != nil {
			return err
		}
	}
	enc := strings.TrimSpace(machine.SftpEncoding)
	if enc == "" {
		machine.SftpEncoding = "auto"
	} else {
		machine.SftpEncoding = enc
	}
	return nil
}
