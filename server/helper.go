package server

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
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

// parseDuration parses a duration string as seconds
func parseDuration(payload string) (time.Duration, error) {
	if payload == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(payload)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, fmt.Errorf("invalid duration: %s", payload)
	}
	return time.Duration(v) * time.Second, err
}

// jsonDecoder returns a json decoder with disallowed unknown fields
func jsonDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec
}

// omitEmpty returns true if struct field is omitempty
func omitEmpty(f reflect.StructField) bool {
	tag := f.Tag.Get("json")
	values := strings.Split(tag, ",")
	return slices.Contains(values, "omitempty")
}
