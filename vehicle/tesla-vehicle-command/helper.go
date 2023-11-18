package vc

import (
	"strings"

	"github.com/evcc-io/evcc/api"
)

// ApiError converts HTTP 408 error to ErrTimeout
func ApiError(err error) error {
	if err != nil && (strings.HasSuffix(err.Error(), "408 Request Timeout") || strings.HasSuffix(err.Error(), "408 (Request Timeout)")) {
		err = api.ErrAsleep
	}
	return err
}
