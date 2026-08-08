package utils

import "fmt"

// FormatTransferSpeed formats a byte/sec rate with adaptive binary units (B/s → PB/s).
// Values below 1 KiB stay in B/s; otherwise scale at 1024 boundaries (KB/s, MB/s, …).
func FormatTransferSpeed(bytesPerSec float64) string {
	if bytesPerSec < 0 {
		bytesPerSec = 0
	}
	units := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s", "PB/s"}
	v := bytesPerSec
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", v, units[i])
	}
	return fmt.Sprintf("%.2f %s", v, units[i])
}
