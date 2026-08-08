package utils

import "testing"

func TestFormatTransferSpeed(t *testing.T) {
	tests := []struct {
		bps  float64
		want string
	}{
		{0, "0 B/s"},
		{512, "512 B/s"},
		{1023, "1023 B/s"},
		{1024, "1.00 KB/s"},
		{2063.77 * 1024, "2.02 MB/s"}, // 截图中的 KB/s 展示应对应到 MB/s
		{6621.59 * 1024, "6.47 MB/s"},
		{1.5 * 1024 * 1024, "1.50 MB/s"},
		{2 * 1024 * 1024 * 1024, "2.00 GB/s"},
	}
	for _, tt := range tests {
		if got := FormatTransferSpeed(tt.bps); got != tt.want {
			t.Fatalf("FormatTransferSpeed(%v) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}
