package data

import "math"

const (
	DefaultWindowOpacity = 1.0
	MinWindowOpacity     = 0.4
)

// NormalizeWindowOpacity 将透明度限制在 0.4–1；0 或缺省视为不透明。
func NormalizeWindowOpacity(v float64) float64 {
	if v <= 0 || math.IsNaN(v) {
		return DefaultWindowOpacity
	}
	if v < MinWindowOpacity {
		return MinWindowOpacity
	}
	if v > 1 {
		return 1
	}
	return math.Round(v*100) / 100
}
