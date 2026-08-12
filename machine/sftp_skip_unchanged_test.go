package machine

import (
	"testing"
	"time"
)

func TestSameSizeAndMtime(t *testing.T) {
	t0 := time.Unix(1700000000, 123)
	t1 := time.Unix(1700000000, 999)
	t2 := time.Unix(1700000001, 0)
	if !SameSizeAndMtime(10, t0, 10, t1) {
		t.Fatal("same second should match")
	}
	if SameSizeAndMtime(10, t0, 11, t0) {
		t.Fatal("size mismatch")
	}
	if SameSizeAndMtime(10, t0, 10, t2) {
		t.Fatal("mtime mismatch")
	}
}
