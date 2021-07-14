package wrapper

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
)

// go:generate go run ../../cmd/tools/decorate.go -f decorateVehicle -b api.Vehicle -t "api.ChargeState,Status,func() (api.ChargeStatus, error)" -t "api.VehicleFinishTimer,FinishTime,func() (time.Time, error)" -t "api.VehicleRange,Range,func() (int64, error)" -t "api.VehicleClimater,Climater,func() (bool, float64, float64, error)"

// Wrapper wraps an api.Vehicle to capture initialization errors
type Wrapper struct {
	api.Vehicle
	err error
}

// New creates a new Vehicle
func New(w api.Vehicle, err error) (api.Vehicle, error) {
	v := &Wrapper{
		err:     fmt.Errorf("vehicle not available: %w", err),
		Vehicle: w,
	}

	// decorate vehicle with Status
	var status func() (api.ChargeStatus, error)
	if _, ok := w.(api.ChargeState); ok {
		status = v.status
	}

	// decorate vehicle with FinishTimer
	var finishTimer func() (time.Time, error)
	if _, ok := w.(api.VehicleFinishTimer); ok {
		finishTimer = v.finishTimer
	}

	// decorate vehicle with Range
	var rng func() (int64, error)
	if _, ok := w.(api.VehicleRange); ok {
		rng = v.rng
	}

	// decorate vehicle with Range
	var climater func() (bool, float64, float64, error)
	if _, ok := w.(api.VehicleClimater); ok {
		climater = v.climater
	}

	res := decorateVehicle(v, status, finishTimer, rng, climater)

	return res, nil
}

// SoC implements the api.Vehicle interface
func (v *Wrapper) status() (api.ChargeStatus, error) {
	if v.err != nil {
		return api.StatusF, v.err
	}

	return v.Vehicle.(api.ChargeState).Status()
}

// finishTimer implements the api.VehicleFinishTimer interface
func (v *Wrapper) finishTimer() (time.Time, error) {
	if v.err != nil {
		return time.Time{}, v.err
	}

	return v.Vehicle.(api.VehicleFinishTimer).FinishTime()
}

// rng implements the api.VehicleRange interface
func (v *Wrapper) rng() (int64, error) {
	if v.err != nil {
		return 0, v.err
	}

	return v.Vehicle.(api.VehicleRange).Range()
}

// climater implements the api.VehicleClimater interface
func (v *Wrapper) climater() (bool, float64, float64, error) {
	if v.err != nil {
		return false, 0, 0, v.err
	}

	return v.Vehicle.(api.VehicleClimater).Climater()
}
