package data

import (
	"testing"

	"FlashDock/define"
)

// 机器被删掉重建（ID 变了但名称不变）时，历史不能堆积成多条同名记录
func TestRecordConnectMergesRenamedID(t *testing.T) {
	m := &ShellHistoryManager{}
	oldMachine := &define.Machine{ID: "old-id", Name: "demo-host"}
	newMachine := &define.Machine{ID: "new-id", Name: "demo-host"}

	if err := m.RecordConnect(oldMachine, "10.0.0.1", 22, "root"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := m.RecordConnect(oldMachine, "10.0.0.1", 22, "root"); err != nil {
		t.Fatalf("record: %v", err)
	}
	// 机器重建后 ID 变化，但名称相同
	if err := m.RecordConnect(newMachine, "10.0.0.1", 22, "root"); err != nil {
		t.Fatalf("record: %v", err)
	}

	list := m.List()
	if len(list) != 1 {
		t.Fatalf("同名机器应合并为 1 条，实际 %d 条: %+v", len(list), list)
	}
	if list[0].MachineID != "new-id" {
		t.Fatalf("应更新为新 ID，实际 %q", list[0].MachineID)
	}
	if list[0].ConnectCount != 3 {
		t.Fatalf("连接次数应累加为 3，实际 %d", list[0].ConnectCount)
	}
}

// 历史里已经写坏的重复项，List 时要兜底合并
func TestListMergesExistingDuplicates(t *testing.T) {
	m := &ShellHistoryManager{}
	m.records = []define.ShellHistoryRecord{
		{MachineID: "id-a", MachineName: "same-host", Host: "10.0.0.9", Port: 22, User: "root", LastConnectedAt: 100, ConnectCount: 5},
		{MachineID: "id-b", MachineName: "same-host", Host: "10.0.0.9", Port: 22, User: "root", LastConnectedAt: 200, ConnectCount: 7},
		{MachineID: "id-c", MachineName: "same-host", Host: "10.0.0.9", Port: 22, User: "root", LastConnectedAt: 50, ConnectCount: 3},
		{MachineID: "id-d", MachineName: "other-host", Host: "10.0.0.10", Port: 22, User: "root", LastConnectedAt: 150, ConnectCount: 1},
	}
	list := m.List()
	if len(list) != 2 {
		t.Fatalf("应合并为 2 条，实际 %d: %+v", len(list), list)
	}
	// 第一条是 other-host（150）还是 same-host（200）？按时间倒序，same-host 在前
	if list[0].MachineName != "same-host" {
		t.Fatalf("最近连接应是 same-host，实际 %q", list[0].MachineName)
	}
	if list[0].ConnectCount != 15 {
		t.Fatalf("次数应合并为 15，实际 %d", list[0].ConnectCount)
	}
	if list[0].LastConnectedAt != 200 {
		t.Fatalf("应取最近时间 200，实际 %d", list[0].LastConnectedAt)
	}
	if list[0].MachineID != "id-b" {
		t.Fatalf("应保留最近那条的 ID，实际 %q", list[0].MachineID)
	}
}

// 不同机器不应被误合并
func TestListKeepsDistinctMachines(t *testing.T) {
	m := &ShellHistoryManager{}
	m.records = []define.ShellHistoryRecord{
		{MachineID: "1", MachineName: "a", LastConnectedAt: 300, ConnectCount: 1},
		{MachineID: "2", MachineName: "b", LastConnectedAt: 200, ConnectCount: 1},
		{MachineID: "3", MachineName: "c", LastConnectedAt: 100, ConnectCount: 1},
	}
	if got := len(m.List()); got != 3 {
		t.Fatalf("不同机器不应合并，实际 %d 条", got)
	}
}
