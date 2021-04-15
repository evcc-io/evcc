package cloud

import "errors"

var (
	// ErrNotAuthorized indicates request token is not authorized
	ErrNotAuthorized = errors.New("not authorized")

	// ErrVehicleNotAvailable indicates vehicle not available, client should retry to prepare vehicle
	ErrVehicleNotAvailable = errors.New("vehicle not available")
)
