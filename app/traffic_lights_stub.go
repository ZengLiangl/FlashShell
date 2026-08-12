//go:build !darwin

package app

func setTrafficLightPosition(x, y float64) {
	_, _ = x, y
}
