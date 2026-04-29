package templates

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// Duration helpers (numeric-only returns)
// Source: https://github.com/Masterminds/sprig/pull/467
// -----------------------------------------------------------------------------

// asDuration converts common template values into a time.Duration.
//
// Supported inputs:
//   - time.Duration
//   - string duration values parsed by time.ParseDuration (e.g. "1h2m3s")
//   - numeric strings treated as seconds (e.g. "2.5")
//   - ints and uints treated as seconds
//   - floats treated as seconds
func asDuration(v any) (time.Duration, error) {
	switch x := v.(type) {
	case time.Duration:
		return x, nil

	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, fmt.Errorf("empty duration")
		}
		if d, err := time.ParseDuration(s); err == nil {
			return d, nil
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return time.Duration(f * float64(time.Second)), nil
		}
		return 0, fmt.Errorf("could not parse duration %q", x)

	case nil:
		return 0, fmt.Errorf("invalid duration")
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return time.Duration(rv.Int()) * time.Second, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := rv.Uint()
		if u > uint64(math.MaxInt64) {
			return 0, fmt.Errorf("duration seconds overflow: %d", u)
		}
		return time.Duration(int64(u)) * time.Second, nil
	case reflect.Float32, reflect.Float64:
		return time.Duration(rv.Float() * float64(time.Second)), nil
	default:
		return 0, fmt.Errorf("unsupported duration type %T", v)
	}
}

// durationSeconds converts a duration to seconds (float64).
// On error it returns 0.
func durationSeconds(v any) float64 {
	d, err := asDuration(v)
	if err != nil {
		return 0
	}
	return d.Seconds()
}
