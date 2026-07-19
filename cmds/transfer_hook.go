package cmds

import "FlashDock/define"

// TransferReporter 任务上传进度上报（与 Shell 传输队列共用）
type TransferReporter func(rec *define.SftpTransferRecord)

var transferReporter TransferReporter

// SetTransferReporter 注册传输进度上报（由 app 注入）
func SetTransferReporter(r TransferReporter) {
	transferReporter = r
}

func reportTransfer(rec *define.SftpTransferRecord) {
	if transferReporter != nil && rec != nil {
		transferReporter(rec)
	}
}
