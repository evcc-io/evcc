package tesla

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/teslamotors/vehicle-command/pkg/connector/inet"
)

// apiError converts HTTP 408 error to ErrAsleep
// A 408 may carry the Fleet API response body, e.g. "408 Request Timeout: vehicle unavailable: ..."
func apiError(err error) error {
	if err != nil && (errors.Is(err, inet.ErrVehicleNotAwake) ||
		strings.Contains(err.Error(), "408 Request Timeout") || strings.Contains(err.Error(), "408 (Request Timeout)") ||
		strings.Contains(err.Error(), "vehicle is offline or asleep")) {
		err = api.ErrAsleep
	}
	return err
}
