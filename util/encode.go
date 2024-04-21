package util

import (
	"fmt"
	"math"
	"time"
)

// EncodeAny provides a consumer-friendly default encoding for any type
func EncodeAny(v any) any {
	switch val := v.(type) {
	case time.Time:
		if val.IsZero() {
			return nil
		}
		return val.Format(time.RFC3339)

	case time.Duration:
		// must be before stringer to convert to seconds instead of string
		return int(val.Seconds())

	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return nil
		}
		return math.Round(val*1e3) / 1e3

	case fmt.Stringer:
		return val.String()

	default:
		return val
	}
}
