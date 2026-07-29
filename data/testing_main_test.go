package data

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	dir := MustIsolateConfigHomeProcess()
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
