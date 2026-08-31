package data

import (
	"os"
	"testing"

	"FlashDock/crypto"
)

func TestMain(m *testing.M) {
	dir := MustIsolateConfigHomeProcess()
	_ = os.Setenv("FLASH_SHELL_FORCE_FILE_KEYRING", "1")
	if err := crypto.InitVault(); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
