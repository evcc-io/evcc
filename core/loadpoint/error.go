package loadpoint

import (
	"errors"

	"github.com/evcc-io/evcc/api"
)

func AcceptableError(err error) bool {
	return errors.Is(err, api.ErrAsleep) || errors.Is(err, api.ErrMustRetry)
}
