# Shell Mode Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (or implement in-session). Steps use checkbox (`- [ ]`) syntax.

**Goal:** FinalShell-like Shell workspace: history center, machine picker, aux-channel monitor, SFTP file panel.

**Architecture:** Keep existing PTY `ShellSessionPool`. Add `ShellAuxPool` (SSH+SFTP) per machine for monitor + files. Frontend reworks `ShellWorkspace`.

**Tech Stack:** Go (golang.org/x/crypto/ssh, github.com/pkg/sftp), Vue 3, Element Plus, xterm

---

## File map

| File | Role |
|------|------|
| `data/shell_history.go` | Connection history CRUD |
| `machine/shell_aux.go` | Aux SSH: exec monitor cmds + SFTP list |
| `machine/shell_aux_pool.go` | Pool keyed by machine name |
| `define/shell.go` | MonitorSnapshot, SftpEntry DTOs |
| `app/app.go` | Wails APIs + wire aux on connect/disconnect |
| `frontend/.../ShellConnectionHistory.vue` | Center history |
| `frontend/.../ShellMachinePickerDialog.vue` | Folder button dialog |
| `frontend/.../ShellMonitorPanel.vue` | Left monitor |
| `frontend/.../ShellFilePanel.vue` | Bottom SFTP |
| `frontend/.../ShellWorkspace.vue` | Layout orchestration |
| `frontend/.../ShellTerminalTabs.vue` | Folder btn before tabs |
| `frontend/src/composables/useShell.js` | History + monitor + sftp helpers |

---

### Task 1: Connection history backend + APIs

- [ ] Create `data/shell_history.go` (Record: machineId, name, host, port, user, lastConnectedAt, count)
- [ ] App: `GetShellHistory`, `ClearShellHistory`, record on successful `ConnectShell`
- [ ] Verify: `go test` / manual connect updates file

### Task 2: History UI + machine picker

- [ ] `ShellConnectionHistory.vue` — empty-state center list, click → connect
- [ ] `ShellMachinePickerDialog.vue` — grouped machines, connect + settings(edit)
- [ ] `ShellTerminalTabs` / workspace: folder button opens picker
- [ ] When no sessions: show history; else terminals

### Task 3: Aux SSH pool + monitor

- [ ] `ShellAuxManager`: Connect(withSFTP), Exec, ListDir, Close
- [ ] Connect aux when PTY connects; close on disconnect
- [ ] `GetShellMonitor(machineName)` → parse uptime/cpu/mem/top
- [ ] `ShellMonitorPanel.vue` replaces left list; poll active machine 4s
- [ ] Display IP copy, uptime, CPU%, mem%, TOP3

### Task 4: SFTP panel + cd sync

- [ ] APIs: `ListShellFiles`, `GetShellPwd` (exec pwd on aux), `ShellSftpStat`
- [ ] `ShellFilePanel.vue` under terminal: tree + list, hidden toggle
- [ ] On Files open: list cwd; parse terminal input `cd` → refresh; pwd every 2s while open

### Task 5: Wire & polish

- [ ] Edit machine from picker → reuse MachineConfigDialog edit path
- [ ] Leave shell / disconnect cleans aux
- [ ] `wails generate` bindings if needed; smoke test

---

## Out of scope
Disk table, network graphs, multi-tab per host, drag-drop upload (list/browse first; upload can follow)
