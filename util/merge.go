package util

import "dario.cat/mergo"

// Merge merged src into non-pointer dst and returns dst and errors
func Merge[D, S any](dst D, src S, opts ...func(*mergo.Config)) (D, error) {
	err := mergo.Merge(&dst, src, opts...)
	return dst, err
}
