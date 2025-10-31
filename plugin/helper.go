package plugin

import (
	"context"
	"fmt"
	"math"
	"strconv"

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

func contextLogger(ctx context.Context, log *util.Logger) *util.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(util.CtxLogger).(*util.Logger); ok {
			log = l
		}
	}

	return log
}

// parseFloat rejects NaN and Inf values
func parseFloat(payload string) (float64, error) {
	f, err := strconv.ParseFloat(payload, 64)
	if err == nil && (math.IsNaN(f) || math.IsInf(f, 0)) {
		return 0, fmt.Errorf("invalid float value: %s", payload)
	}
	return f, err
}
