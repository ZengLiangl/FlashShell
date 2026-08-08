package app

import (
	"time"

	"FlashDock/define"
)

// reportTaskTransfer 任务 upload 步骤进度上报到 Shell 传输队列
func (a *App) reportTaskTransfer(rec *define.SftpTransferRecord) {
	if rec == nil {
		return
	}
	if rec.ID == "" {
		return
	}
	now := time.Now().Unix()
	if rec.StartedAt == 0 {
		rec.StartedAt = now
	}
	rec.UpdatedAt = now
	if rec.Status == "done" || rec.Status == "error" {
		rec.FinishedAt = now
		a.upsertTransfer(rec)
		return
	}
	// running：异步推送，避免 Wails IPC 阻塞 SFTP 热路径
	store := a.ensureTransferStore()
	store.mu.Lock()
	store.items[rec.ID] = rec
	cp := *rec
	store.mu.Unlock()
	go a.emitTransfer(&cp)
}
