package polestar

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/vehicle/polestar/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpcStatus(t *testing.T) {
	tc := []struct {
		name       string
		connection pb.ChargerConnectionStatus
		charging   pb.ChargingStatus
		expected   api.ChargeStatus
	}{
		{"disconnected", pb.ChargerConnectionStatus_CHARGER_CONNECTION_STATUS_DISCONNECTED, pb.ChargingStatus_CHARGING_STATUS_IDLE, api.StatusA},
		{"unspecified", pb.ChargerConnectionStatus_CHARGER_CONNECTION_STATUS_UNSPECIFIED, pb.ChargingStatus_CHARGING_STATUS_UNSPECIFIED, api.StatusA},
		{"connected idle", pb.ChargerConnectionStatus_CHARGER_CONNECTION_STATUS_CONNECTED, pb.ChargingStatus_CHARGING_STATUS_IDLE, api.StatusB},
		{"connected done", pb.ChargerConnectionStatus_CHARGER_CONNECTION_STATUS_CONNECTED, pb.ChargingStatus_CHARGING_STATUS_DONE, api.StatusB},
		{"charging", pb.ChargerConnectionStatus_CHARGER_CONNECTION_STATUS_CONNECTED, pb.ChargingStatus_CHARGING_STATUS_CHARGING, api.StatusC},
		{"smart charging", pb.ChargerConnectionStatus_CHARGER_CONNECTION_STATUS_CONNECTED, pb.ChargingStatus_CHARGING_STATUS_SMART_CHARGING, api.StatusC},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			v := &GrpcProvider{
				batteryG: func() (*pb.Battery, error) {
					return &pb.Battery{
						ChargerConnectionStatus: tc.connection,
						ChargingStatus:          tc.charging,
					}, nil
				},
			}

			status, err := v.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, status)
		})
	}
}

func TestGrpcFinishTime(t *testing.T) {
	captured := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)

	v := &GrpcProvider{
		batteryG: func() (*pb.Battery, error) {
			return &pb.Battery{
				EstimatedChargingTimeToFullMinutes: 30,
				Timestamp:                          &pb.Timestamp{Seconds: captured.Unix()},
			}, nil
		},
	}

	// finish time is anchored to the API capture timestamp, not time.Now()
	ft, err := v.FinishTime()
	require.NoError(t, err)
	assert.True(t, captured.Add(30*time.Minute).Equal(ft))
}

func TestGrpcFinishTimeNotCharging(t *testing.T) {
	v := &GrpcProvider{
		batteryG: func() (*pb.Battery, error) {
			return &pb.Battery{EstimatedChargingTimeToFullMinutes: 0}, nil
		},
	}

	_, err := v.FinishTime()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}
