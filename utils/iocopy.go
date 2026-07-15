package utils

import (
	"io"
	"sync"
)

// TransferBufferSize is the preferred I/O chunk for network copies (SFTP/HTTP).
// Go's default io.Copy uses 32KiB which under-utilizes bandwidth on modern links.
const TransferBufferSize = 512 * 1024

var transferBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, TransferBufferSize)
		return &b
	},
}

// CopyBuffer copies src to dst using a pooled 512KiB buffer.
func CopyBuffer(dst io.Writer, src io.Reader) (int64, error) {
	bp := transferBufPool.Get().(*[]byte)
	defer transferBufPool.Put(bp)
	return io.CopyBuffer(dst, src, *bp)
}
