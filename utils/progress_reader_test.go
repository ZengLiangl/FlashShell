package utils

import (
	"bytes"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func TestCountingReader_ReportsThrottledProgress(t *testing.T) {
	var calls atomic.Int32
	r := NewCountingReader(bytes.NewReader(bytes.Repeat([]byte("a"), 3000)), 3000, 0, func(transferred, total int64, _ float64) {
		calls.Add(1)
		if total != 3000 {
			t.Errorf("total=%d", total)
		}
		if transferred <= 0 || transferred > total {
			t.Errorf("transferred=%d", transferred)
		}
	})

	buf := make([]byte, 1000)
	for {
		_, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		// Keep reads within throttle window so only EOF/completion forces extra report.
		time.Sleep(10 * time.Millisecond)
	}

	if calls.Load() < 1 {
		t.Fatal("expected at least one progress callback")
	}
	// 3 reads of 1000 within <400ms + final => should not report every read.
	if calls.Load() > 3 {
		t.Fatalf("progress called too often: %d", calls.Load())
	}
}
