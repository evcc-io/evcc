package util

// PtrTo returns a pointer to the value passed as argument. The zero is returned as nil.
func PtrTo[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// PtrToWithZero returns a pointer to the value passed as argument, including its zero value.
func PtrToWithZero[T any](v T) *T {
	return &v
}
