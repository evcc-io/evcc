package server

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
)

// pass converts a simple api without return value to api with nil error return value
func pass[T any](f func(T)) func(T) error {
	return func(v T) error {
		f(v)
		return nil
	}
}

// parseFloat rejects NaN and Inf values
func parseFloat(payload string) (float64, error) {
	f, err := strconv.ParseFloat(payload, 64)
	if err == nil && (math.IsNaN(f) || math.IsInf(f, 0)) {
		err = fmt.Errorf("invalid float value: %s", payload)
	}
	return f, err
}

// ptr returns a pointer to the given value or nil for the zero value
func ptr[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// ptrZero returns a pointer to the given value, including the zero value
func ptrZero[T comparable](v T) *T {
	return &v
}

// jsonDecoder returns a json decoder with disallowed unknown fields
func jsonDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec
}
