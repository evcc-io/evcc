package myskoda

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVehicleResponse(t *testing.T) {
	sample := `{
		"vehicle": {
			"vin": "TMBJB9NY5RF999999",
			"name": "My Enyaq",
			"odometer": { "mileageInKm": 12753 },
			"airConditioning": { "state": "HEATING" },
			"charging": {
				"isVehicleInSavedLocation": true,
				"status": {
					"chargePowerInKw": 20.16,
					"remainingTimeToFullyChargedInMinutes": 15,
					"state": "CHARGING",
					"chargeType": "AC",
					"battery": {
						"remainingCruisingRangeInMeters": 249000,
						"stateOfChargeInPercent": 71
					}
				},
				"settings": { "targetStateOfChargeInPercent": 80 }
			}
		},
		"errors": []
	}`

	var res VehicleResponse
	require.NoError(t, json.Unmarshal([]byte(sample), &res))

	v := &Provider{dataG: func() (VehicleResponse, error) { return res, nil }}

	soc, err := v.Soc()
	require.NoError(t, err)
	assert.Equal(t, 71.0, soc)

	status, err := v.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	rng, err := v.Range()
	require.NoError(t, err)
	assert.Equal(t, int64(249), rng)

	odo, err := v.Odometer()
	require.NoError(t, err)
	assert.Equal(t, 12753.0, odo)

	limit, err := v.GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(80), limit)

	climate, err := v.Climater()
	require.NoError(t, err)
	assert.True(t, climate)
}

func TestStatus(t *testing.T) {
	statusG := func(state string) *Provider {
		res := VehicleResponse{Vehicle: Vehicle{Charging: &Charging{Status: &ChargingStatus{State: state}}}}
		return &Provider{dataG: func() (VehicleResponse, error) { return res, nil }}
	}

	for _, tc := range []struct {
		state  string
		status api.ChargeStatus
	}{
		{"CONNECT_CABLE", api.StatusA},
		{"READY_FOR_CHARGING", api.StatusB},
		{"CHARGING_INTERRUPTED", api.StatusB},
		{"DISCHARGING", api.StatusB},
		{"CHARGING", api.StatusC},
		{"CONSERVING", api.StatusC},
	} {
		status, err := statusG(tc.state).Status()
		require.NoError(t, err, tc.state)
		assert.Equal(t, tc.status, status, tc.state)
	}

	// an unknown state must not read as plugged in, that would assign the vehicle
	_, err := statusG("SOMETHING_NEW").Status()
	assert.Error(t, err)
}

func TestVehicleResponsePartErrors(t *testing.T) {
	sample := `{
		"vehicle": { "vin": "TMBJB9NY5RF999999" },
		"errors": [{ "type": "CHARGING_UNAVAILABLE", "description": "Charging status could not be retrieved." }]
	}`

	var res VehicleResponse
	require.NoError(t, json.Unmarshal([]byte(sample), &res))

	v := &Provider{dataG: func() (VehicleResponse, error) { return res, nil }}

	_, err := v.Soc()
	assert.ErrorContains(t, err, "CHARGING_UNAVAILABLE")
	// the description must not be logged on every update
	assert.ErrorIs(t, err, api.ErrNotAvailable)

	_, err = v.Odometer()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}
