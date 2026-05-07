package bluelink

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestVehicleStatusRangeHonorsUnit(t *testing.T) {
	status := VehicleStatus{
		EvStatus: &struct {
			BatteryCharge bool
			BatteryStatus float64
			BatteryPlugin int
			RemainTime2   struct {
				Atc struct {
					Value int
					Unit  int
				}
			}
			ChargePortDoorOpenStatus int
			DrvDistance              []DrivingDistance
			ReservChargeInfos        ReservChargeInfo
		}{
			DrvDistance: []DrivingDistance{
				{
					RangeByFuel: struct {
						EvModeRange struct {
							Value float64
							Unit  int
						}
					}{
						EvModeRange: struct {
							Value float64
							Unit  int
						}{
							Value: 168.38834951456312,
							Unit:  unitMiles,
						},
					},
				},
			},
		},
	}

	got, err := status.Range()
	require.NoError(t, err)
	require.EqualValues(t, 271, got)
}

func TestVehicleStatusRangeKilometersUnchanged(t *testing.T) {
	status := VehicleStatus{
		EvStatus: &struct {
			BatteryCharge bool
			BatteryStatus float64
			BatteryPlugin int
			RemainTime2   struct {
				Atc struct {
					Value int
					Unit  int
				}
			}
			ChargePortDoorOpenStatus int
			DrvDistance              []DrivingDistance
			ReservChargeInfos        ReservChargeInfo
		}{
			DrvDistance: []DrivingDistance{
				{
					RangeByFuel: struct {
						EvModeRange struct {
							Value float64
							Unit  int
						}
					}{
						EvModeRange: struct {
							Value float64
							Unit  int
						}{
							Value: 168.9,
							Unit:  1,
						},
					},
				},
			},
		},
	}

	got, err := status.Range()
	require.NoError(t, err)
	require.EqualValues(t, 168, got)
}

func TestVehicleStatusRangeNotAvailable(t *testing.T) {
	got, err := VehicleStatus{}.Range()
	require.ErrorIs(t, err, api.ErrNotAvailable)
	require.Zero(t, got)
}
