package util

// Map maps a given slice using func
func Map[From, To any](s []From, f func(From) To) []To {
	res := make([]To, len(s))
	for i, v := range s {
		res[i] = f(v)
	}
	return res
}

// Filter filters a given slice by func
func Filter[T any](tt []T, f func(t T) bool) []T {
	res := make([]T, 0)
	for _, t := range tt {
		if f(t) {
			res = append(res, t)
		}
	}
	return res
}
