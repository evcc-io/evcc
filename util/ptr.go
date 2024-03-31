package util

// PtrTo returns a pointer to the value passed as argument.
func PtrTo[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
