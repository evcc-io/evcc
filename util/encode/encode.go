package encode

import (
	"fmt"
	"math"
	"time"
)

type encoder struct {
	withDuration bool
}

type Encoder interface {
	Encode(v any) any
}

type EncoderOption func(*encoder)

// NewEncoder creates a new encoder with the following default conversions:
// - float NaN/Inf are converted to nil
// - zero time.Time are converted to nil
// - durations are converted to seconds using WithDuration()
// - fmt.Stringer are converted to string
// - time.Time are converted to RFC3339 string
func NewEncoder(opt ...EncoderOption) Encoder {
	res := new(encoder)
	for _, o := range opt {
		o(res)
	}
	return res
}

func WithDuration() EncoderOption {
	return func(e *encoder) {
		e.withDuration = true
	}
}

// Encode provides a consumer-friendly default encoding for any type
func (e encoder) Encode(v any) any {
	switch val := v.(type) {
	case time.Time:
		if val.IsZero() {
			return nil
		}
		return val.Format(time.RFC3339)

	case time.Duration:
		// must be before stringer to convert to seconds instead of string
		if e.withDuration {
			return int(val.Seconds())
		}
		return val

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
