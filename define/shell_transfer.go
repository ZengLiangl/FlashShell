package define

import "errors"

// ErrTransferPaused 用户暂停/取消传输
var ErrTransferPaused = errors.New("传输已暂停")

// SftpTransferRecord 文件传输记录
type SftpTransferRecord struct {
	ID          string  `json:"id"`
	MachineName string  `json:"machineName"`
	Direction   string  `json:"direction"` // download | upload
	Name        string  `json:"name"`
	LocalPath   string  `json:"localPath"`
	RemotePath  string  `json:"remotePath"`
	IsDir       bool    `json:"isDir"`
	Status      string  `json:"status"` // pending | running | done | error | paused
	Total       int64   `json:"total"`
	Transferred int64   `json:"transferred"`
	Percent     float64 `json:"percent"`
	SpeedBPS    float64 `json:"speedBps"`
	Error       string  `json:"error,omitempty"`
	StartedAt   int64   `json:"startedAt"`
	UpdatedAt   int64   `json:"updatedAt"`
	FinishedAt  int64   `json:"finishedAt,omitempty"`
}
