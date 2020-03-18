package core

// min calculates minimum of two integer values
func min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// max calculates maximum of two integer values
func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

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
