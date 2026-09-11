package livewire

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringFloat(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
		err  bool
	}{
		{`"87"`, 87, false},
		{`"87.5"`, 87.5, false},
		{`"87%"`, 87, false},
		{`87`, 87, false},
		{`null`, 0, false},
		{`""`, 0, false},
		{`"abc"`, 0, true},
	} {
		var f StringFloat
		err := json.Unmarshal([]byte(tc.in), &f)
		if tc.err {
			assert.Error(t, err, tc.in)
			continue
		}
		require.NoError(t, err, tc.in)
		assert.Equal(t, tc.want, float64(f), tc.in)
	}
}

func TestChargingStatusResponse(t *testing.T) {
	flat := `{"chargingStatus":true,"pluggedIn":true,"batteryPercentage":"87","range":120,"timeToMaxLimit":45,"maxLimit":90,"odometer":1234.5}`
	wrapped := `{"bikeChargingData":` + flat + `}`

	for _, body := range []string{flat, wrapped} {
		var res ChargingStatusResponse
		require.NoError(t, json.Unmarshal([]byte(body), &res))
		require.NoError(t, res.Err())

		data := res.Data()
		assert.True(t, data.ChargingStatus)
		assert.True(t, data.PluggedIn)
		assert.Equal(t, 87.0, float64(data.BatteryPercentage))
		assert.Equal(t, 120.0, data.Range)
		assert.Equal(t, int64(45), data.TimeToMaxLimit)
		assert.Equal(t, int64(90), data.MaxLimit)
		assert.Equal(t, 1234.5, data.Odometer)
	}
}

func TestErrorEnvelope(t *testing.T) {
	var res ChargingStatusResponse
	require.NoError(t, json.Unmarshal([]byte(`{"error":{"code":"3000","description":"Command manager api failed"}}`), &res))

	err := res.Err()
	require.Error(t, err)
	assert.Equal(t, "Command manager api failed (3000)", err.Error())
}

func TestBikesResponse(t *testing.T) {
	var arr BikesResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"id":"1","vin":"VIN1"}]`), &arr))
	require.Len(t, arr.Bikes, 1)
	assert.Equal(t, "VIN1", arr.Bikes[0].VIN)

	var obj BikesResponse
	require.NoError(t, json.Unmarshal([]byte(`{"bikes":[{"id":"2","vin":"VIN2"}]}`), &obj))
	require.Len(t, obj.Bikes, 1)
	assert.Equal(t, "2", obj.Bikes[0].ID)

	var errRes BikesResponse
	require.NoError(t, json.Unmarshal([]byte(`{"error":{"code":"1","description":"nope"}}`), &errRes))
	assert.Error(t, errRes.Err())
}
