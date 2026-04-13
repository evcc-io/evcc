//go:build !linux

package sponsor

// checkHemsPro checks if the hardware is a supported HEMS Pro device and returns sponsor subject
func checkHemsPro() string {
	return ""
}
