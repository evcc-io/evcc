package util

// SignFromPower is a helper function to create signed current from signed power bypassing already signed current
func SignFromPower(current float64, power float64) float64 {
	if current > 0 && power < 0 {
		return -current
	}
	return current
}
