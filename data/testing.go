package data

import (
	"os"
	"testing"
)

// ResetAppDataCacheForTest 清空 app_data 进程内缓存（测试切换配置目录时调用）
func ResetAppDataCacheForTest() {
	appDataMu.Lock()
	defer appDataMu.Unlock()
	appDataCache = nil
	appDataLoaded = false
}

// IsolateConfigHome 将 FLASHDOCK_CONFIG_HOME 指到临时目录，避免测试写真实 ~/.flashdock。
// 返回隔离后的配置目录路径。
func IsolateConfigHome(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(ConfigHomeEnv, dir)
	ResetAppDataCacheForTest()
	t.Cleanup(ResetAppDataCacheForTest)
	return dir
}

// MustIsolateConfigHomeProcess 供 TestMain 使用：进程级隔离配置目录。
// 调用方负责在测试结束后删除返回的目录。
func MustIsolateConfigHomeProcess() string {
	dir, err := os.MkdirTemp("", "flashdock-testhome-*")
	if err != nil {
		panic("创建测试配置目录失败: " + err.Error())
	}
	if err := os.Setenv(ConfigHomeEnv, dir); err != nil {
		panic("设置 FLASHDOCK_CONFIG_HOME 失败: " + err.Error())
	}
	ResetAppDataCacheForTest()
	return dir
}
