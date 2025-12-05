//go:build !linux

package sponsor

const hemspro = "hemspro"

// checkHemsPro checks if the hardware is a supported HEMS Pro device and returns sponsor subject
func checkHemsPro() string {
	return ""
}
