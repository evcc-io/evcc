package polestar

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/polestar/pb"
)

// GrpcProvider implements the vehicle API using the Polestar gRPC battery
// service for charge state and the GraphQL API for odometer.
type GrpcProvider struct {
	batteryG  func() (*pb.Battery, error)
	odometerG func() (float64, error)
}

// NewGrpcProvider creates a Polestar gRPC vehicle data provider
func NewGrpcProvider(grpc *GrpcAPI, api *API, vin string, timeout, cache time.Duration) *GrpcProvider {
	return &GrpcProvider{
		batteryG: util.Cached(func() (*pb.Battery, error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			return grpc.Battery(ctx, vin)
		}, cache),
		odometerG: util.Cached(func() (float64, error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			return api.Odometer(ctx, vin)
		}, cache),
	}
}

var _ api.Battery = (*GrpcProvider)(nil)

// Soc implements the api.Battery interface
func (v *GrpcProvider) Soc() (float64, error) {
	res, err := v.batteryG()
	if err != nil {
		return 0, err
	}
	return res.GetBatteryChargeLevelPercentage(), nil
}

var _ api.ChargeState = (*GrpcProvider)(nil)

// Status implements the api.ChargeState interface
func (v *GrpcProvider) Status() (api.ChargeStatus, error) {
	res, err := v.batteryG()
	if err != nil {
		return api.StatusNone, err
	}

	switch res.GetChargingStatus() {
	case pb.ChargingStatus_CHARGING_STATUS_CHARGING, pb.ChargingStatus_CHARGING_STATUS_SMART_CHARGING:
		return api.StatusC, nil
	}

	if res.GetChargerConnectionStatus() == pb.ChargerConnectionStatus_CHARGER_CONNECTION_STATUS_CONNECTED {
		return api.StatusB, nil
	}

	return api.StatusA, nil
}

var _ api.VehicleRange = (*GrpcProvider)(nil)

// Range implements the api.VehicleRange interface
func (v *GrpcProvider) Range() (int64, error) {
	res, err := v.batteryG()
	if err != nil {
		return 0, err
	}
	return int64(res.GetEstimatedDistanceToEmptyKm()), nil
}

var _ api.VehicleFinishTimer = (*GrpcProvider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *GrpcProvider) FinishTime() (time.Time, error) {
	res, err := v.batteryG()
	if err != nil {
		return time.Time{}, err
	}

	minutes := res.GetEstimatedChargingTimeToFullMinutes()
	if minutes <= 0 {
		return time.Time{}, api.ErrNotAvailable
	}

	// anchor the relative remaining time to the API's capture timestamp
	base := time.Now()
	if ts := res.GetTimestamp(); ts.GetSeconds() > 0 {
		base = time.Unix(ts.GetSeconds(), int64(ts.GetNanos()))
	}

	return base.Add(time.Duration(minutes) * time.Minute), nil
}

var _ api.VehicleOdometer = (*GrpcProvider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *GrpcProvider) Odometer() (float64, error) {
	return v.odometerG()
}
