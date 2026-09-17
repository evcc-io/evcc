package polestar

import (
	"strconv"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	tc := []struct {
		name       string
		connection string
		charging   string
		expected   api.ChargeStatus
	}{
		{"disconnected", "CHARGER_CONNECTION_STATUS_DISCONNECTED", "CHARGING_STATUS_IDLE", api.StatusA},
		{"unspecified", "CHARGER_CONNECTION_STATUS_UNSPECIFIED", "CHARGING_STATUS_UNSPECIFIED", api.StatusA},
		{"connected idle", "CHARGER_CONNECTION_STATUS_CONNECTED", "CHARGING_STATUS_IDLE", api.StatusB},
		{"connected done", "CHARGER_CONNECTION_STATUS_CONNECTED", "CHARGING_STATUS_DONE", api.StatusB},
		{"charging", "CHARGER_CONNECTION_STATUS_CONNECTED", "CHARGING_STATUS_CHARGING", api.StatusC},
		{"smart charging", "CHARGER_CONNECTION_STATUS_CONNECTED", "CHARGING_STATUS_SMART_CHARGING", api.StatusC},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			v := &Provider{
				batteryG: func() (Battery, error) {
					return Battery{
						ChargerConnectionStatus: tc.connection,
						ChargingStatusV2:        tc.charging,
					}, nil
				},
			}

			status, err := v.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, status)
		})
	}
}

func TestSoc(t *testing.T) {
	v := &Provider{
		batteryG: func() (Battery, error) {
			return Battery{BatteryChargeLevelPercentage: 42}, nil
		},
	}

	soc, err := v.Soc()
	require.NoError(t, err)
	assert.Equal(t, 42.0, soc)
}

func TestRange(t *testing.T) {
	v := &Provider{
		batteryG: func() (Battery, error) {
			return Battery{EstimatedDistanceToEmptyKm: 321}, nil
		},
	}

	rng, err := v.Range()
	require.NoError(t, err)
	assert.Equal(t, int64(321), rng)
}

func TestOdometer(t *testing.T) {
	v := &Provider{
		odometerG: func() (Odometer, error) {
			return Odometer{OdometerMeters: 123456}, nil
		},
	}

	odo, err := v.Odometer()
	require.NoError(t, err)
	assert.Equal(t, 123.456, odo)
}

func TestLimitSoc(t *testing.T) {
	v := &Provider{
		targetSocG: func() (TargetSoc, error) {
			var res TargetSoc
			res.TargetSoc.BatteryChargeTargetLevel = 80
			return res, nil
		},
	}

	limit, err := v.GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(80), limit)
}

func TestFinishTime(t *testing.T) {
	captured := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)

	v := &Provider{
		batteryG: func() (Battery, error) {
			return Battery{
				EstimatedChargingTimeToFullMinutes: 30,
				Timestamp:                          Timestamp{Seconds: strconv.FormatInt(captured.Unix(), 10)},
			}, nil
		},
	}

	// finish time is anchored to the API capture timestamp, not time.Now()
	ft, err := v.FinishTime()
	require.NoError(t, err)
	assert.True(t, captured.Add(30*time.Minute).Equal(ft))
}

func TestFinishTimeNotCharging(t *testing.T) {
	v := &Provider{
		batteryG: func() (Battery, error) {
			return Battery{EstimatedChargingTimeToFullMinutes: 0}, nil
		},
	}

	_, err := v.FinishTime()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}
