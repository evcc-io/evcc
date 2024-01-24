package vc

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/teslamotors/vehicle-command/pkg/connector/inet"
)

// ApiError converts HTTP 408 error to ErrTimeout
func ApiError(err error) error {
	if err != nil && (errors.Is(err, inet.ErrVehicleNotAwake) ||
		strings.HasSuffix(err.Error(), "408 Request Timeout") || strings.HasSuffix(err.Error(), "408 (Request Timeout)")) {
		err = api.ErrAsleep
	}
	return err
}
