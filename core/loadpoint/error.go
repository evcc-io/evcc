package loadpoint

import (
	"errors"

	"github.com/evcc-io/evcc/api"
)

func AcceptableError(err error) bool {
	for _, e := range []error{api.ErrAsleep, api.ErrMustRetry, api.ErrNotAvailable} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
