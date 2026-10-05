package pleos

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	tc := []struct {
		name     string
		plugin   string
		status   string
		charging bool
		expected api.ChargeStatus
	}{
		{"disconnected", "disconnected", "notCharging", false, api.StatusA},
		{"invalid", "invalid", "notCharging", false, api.StatusA},
		{"connected", "connected", "notCharging", false, api.StatusB},
		{"reserved", "connected", "reservedCharging", false, api.StatusB},
		{"charging", "connected", "charging", true, api.StatusC},
		{"fast charging", "connected", "fastCharging", true, api.StatusC},
		{"charging flag only", "connected", "notCharging", true, api.StatusC},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			v := &Provider{
				batteryG: func() (Battery, error) {
					var res Battery
					res.Charge.Plugin = tc.plugin
					res.Charge.Status = tc.status
					res.Charge.Charging = tc.charging
					return res, nil
				},
			}

			status, err := v.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, status)
		})
	}
}

func TestBattery(t *testing.T) {
	// sample from https://github.com/evcc-io/evcc/issues/34339
	const sample = `{"charge":{"plugin":"connected","status":"notCharging","charging":false,"stateOfCharge":65.0,
		"targetStateOfCharge":{"standard":90,"quick":80},"remainTime":100},"timestamp":"2026-10-02T13:51:37Z"}`

	var res Battery
	require.NoError(t, json.Unmarshal([]byte(sample), &res))

	v := &Provider{batteryG: func() (Battery, error) { return res, nil }}

	soc, err := v.Soc()
	require.NoError(t, err)
	assert.Equal(t, 65.0, soc)

	limit, err := v.GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(90), limit)

	status, err := v.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusB, status)
}

func TestRange(t *testing.T) {
	const sample = `{"distanceToEmpties":[{"type":"ICE","value":"invalid","unit":"km"},{"type":"EV","value":"106","unit":"miles"}]}`

	var res Powertrain
	require.NoError(t, json.Unmarshal([]byte(sample), &res))

	v := &Provider{powertrainG: func() (Powertrain, error) { return res, nil }}

	rng, err := v.Range()
	require.NoError(t, err)
	assert.Equal(t, int64(170), rng)

	v.powertrainG = func() (Powertrain, error) { return Powertrain{}, nil }
	_, err = v.Range()
	require.ErrorIs(t, err, api.ErrNotAvailable)

	// live response of an Ioniq 5 without reported range
	require.NoError(t, json.Unmarshal([]byte(`{"distanceToEmpties":[{"type":"EV","value":"-","unit":"km"}]}`), &res))
	v.powertrainG = func() (Powertrain, error) { return res, nil }
	_, err = v.Range()
	require.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestOdometer(t *testing.T) {
	v := &Provider{drivingG: func() (Driving, error) {
		return Driving{Odometer: Distance{Value: 6549.2, Unit: "km"}}, nil
	}}

	odo, err := v.Odometer()
	require.NoError(t, err)
	assert.Equal(t, 6549.2, odo)

	v.drivingG = func() (Driving, error) {
		return Driving{Odometer: Distance{Value: 1500, Unit: "meter"}}, nil
	}
	odo, err = v.Odometer()
	require.NoError(t, err)
	assert.Equal(t, 1.5, odo)
}
