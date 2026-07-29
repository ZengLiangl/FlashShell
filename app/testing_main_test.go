package app

import (
	"os"
	"testing"

	"FlashDock/data"
)

func TestMain(m *testing.M) {
	dir := data.MustIsolateConfigHomeProcess()
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
