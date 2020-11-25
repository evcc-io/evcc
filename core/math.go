package core

// clamp calculates minimum of two integer values
func clamp(x, min, max int64) int64 {
	if x <= min {
		return min
	}
	if x >= max {
		return max
	}
	return x
}
