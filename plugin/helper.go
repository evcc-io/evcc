package plugin

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// setFormattedValue formats a message template or returns the value formatted as %v if the message template is empty
func setFormattedValue(message, param string, v any) (string, error) {
	if message == "" {
		return fmt.Sprintf("%v", v), nil
	}

	return util.ReplaceFormatted(message, map[string]any{
		param: v,
	})
}

// knownErrors maps string responses to known error codes
func knownErrors(b []byte) error {
	switch string(b) {
	case "ErrAsleep":
		return api.ErrAsleep
	case "ErrMustRetry":
		return api.ErrMustRetry
	case "ErrNotAvailable":
		return api.ErrNotAvailable
	default:
		return nil
	}
}

// parseFloat rejects NaN and Inf values
func parseFloat(payload string) (float64, error) {
	f, err := strconv.ParseFloat(payload, 64)
	if err == nil && (math.IsNaN(f) || math.IsInf(f, 0)) {
		return 0, fmt.Errorf("invalid float value: %s", payload)
	}
	return f, err
}

// unixThreshold is the Unix timestamp for 2026-01-01 00:00:00 UTC, used to distinguish
// Unix timestamps from duration-in-seconds when parsing numeric finish times.
const unixThreshold = 1767225600

// parseRelativeTime parses a string into an absolute time.Time.
// Supported formats:
//   - RFC3339 timestamp (e.g. "2026-05-03T14:00:00Z") → interpreted as absolute time
//   - Go duration string (e.g. "1h30m") → interpreted as remaining duration, added to time.Now()
//   - Numeric string ≥ 1767225600 (e.g. "1767225600") → interpreted as Unix timestamp (absolute time)
//   - Numeric string < 1767225600 (e.g. "5400") → interpreted as remaining seconds, added to time.Now()
//
// For relative formats, time.Now() is evaluated at the time of each call, providing a
// fresh estimate. This matches the behavior of hardcoded charger/vehicle implementations.
func parseRelativeTime(s string) (time.Time, error) {
	// Try RFC3339 timestamp first (absolute time)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try Go duration string (relative)
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(d), nil
	}

	// Try numeric: Unix timestamp (absolute) or duration in seconds (relative)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f >= unixThreshold {
			return time.Unix(int64(f), 0), nil
		}
		return time.Now().Add(time.Duration(f * float64(time.Second))), nil
	}

	return time.Time{}, fmt.Errorf("invalid finish time: %s", s)
}
