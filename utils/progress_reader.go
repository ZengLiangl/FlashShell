package utils

import (
	"io"
	"time"
)

// ProgressFunc reports transfer progress. speedBPS is a recent window estimate (bytes/sec).
type ProgressFunc func(transferred, total int64, speedBPS float64)

// CountingReader wraps a reader and throttles progress callbacks (~400ms).
// Size() returns remaining bytes so pkg/sftp.File.ReadFrom can size concurrency.
type CountingReader struct {
	r           io.Reader
	total       int64
	transferred int64
	onProgress  ProgressFunc
	lastReport  time.Time
	windowStart time.Time
	windowBytes int64
	speedBPS    float64
}

// NewCountingReader starts counting at alreadyTransferred (e.g. resume offset).
func NewCountingReader(r io.Reader, total, alreadyTransferred int64, onProgress ProgressFunc) *CountingReader {
	return &CountingReader{
		r:           r,
		total:       total,
		transferred: alreadyTransferred,
		onProgress:  onProgress,
	}
}

// Size implements the size hint used by pkg/sftp concurrent ReadFrom.
func (c *CountingReader) Size() int64 {
	remain := c.total - c.transferred
	if remain < 0 {
		return 0
	}
	return remain
}

// Transferred returns bytes accounted so far (including resume offset).
func (c *CountingReader) Transferred() int64 {
	return c.transferred
}

// SpeedBPS returns the latest windowed speed estimate.
func (c *CountingReader) SpeedBPS() float64 {
	return c.speedBPS
}

func (c *CountingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.transferred += int64(n)
		c.windowBytes += int64(n)
		now := time.Now()
		if c.windowStart.IsZero() {
			c.windowStart = now
		}
		elapsed := now.Sub(c.windowStart).Seconds()
		if elapsed >= 0.4 {
			c.speedBPS = float64(c.windowBytes) / elapsed
			c.windowStart = now
			c.windowBytes = 0
		}
		done := err == io.EOF || (c.total > 0 && c.transferred >= c.total)
		if c.onProgress != nil && (now.Sub(c.lastReport) >= 400*time.Millisecond || done) {
			c.lastReport = now
			c.onProgress(c.transferred, c.total, c.speedBPS)
		}
	} else if err == io.EOF && c.onProgress != nil {
		c.onProgress(c.transferred, c.total, c.speedBPS)
	}
	return n, err
}
