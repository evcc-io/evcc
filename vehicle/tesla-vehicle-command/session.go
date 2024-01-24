package vc

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/universalmessage"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
)

type CommandSession struct {
	vehicle *vehicle.Vehicle
	timeout time.Duration
}

func NewCommandSession(vv *vehicle.Vehicle, timeout time.Duration) (*CommandSession, error) {
	if err := vv.Connect(context.Background()); err != nil {
		return nil, err
	}

	if err := vv.StartSession(context.Background(), []universalmessage.Domain{universalmessage.Domain_DOMAIN_INFOTAINMENT}); err != nil {
		return nil, err
	}

	v := &CommandSession{
		vehicle: vv,
		timeout: timeout,
	}

	return v, nil
}

var _ api.CurrentController = (*CommandSession)(nil)

// MaxCurrent implements the api.CurrentController interface
func (v *CommandSession) MaxCurrent(current int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	return apiError(v.vehicle.SetChargingAmps(ctx, int32(current)))
}

var _ api.Resurrector = (*CommandSession)(nil)

// WakeUp implements the api.Resurrector interface
func (v *CommandSession) WakeUp() error {
	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	return apiError(v.vehicle.Wakeup(ctx))
}

var _ api.VehicleChargeController = (*CommandSession)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *CommandSession) StartCharge() error {
	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	err := apiError(v.vehicle.ChargeStart(ctx))

	// ignore charging or complete
	if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
		err = nil
	}

	return err
}

// StopCharge implements the api.VehicleChargeController interface
func (v *CommandSession) StopCharge() error {
	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	err := apiError(v.vehicle.ChargeStop(ctx))

	// ignore sleeping vehicle
	if errors.Is(err, api.ErrAsleep) {
		err = nil
	}

	return err
}
