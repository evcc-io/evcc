package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// setFormattedValue formats a message template or returns the value formatted as %v if the message template is empty
func setFormattedValue(message, param string, v interface{}) (string, error) {
	if message == "" {
		return fmt.Sprintf("%v", v), nil
	}

	return util.ReplaceFormatted(message, map[string]interface{}{
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
