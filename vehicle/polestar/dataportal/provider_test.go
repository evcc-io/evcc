package dataportal

import (
	"testing"

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
