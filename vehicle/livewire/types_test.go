package livewire

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringFloat(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
		err  bool
	}{
		{`"4"`, 4, false},
		{`"0.0"`, 0, false},
		{`"87.5"`, 87.5, false},
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
	var res ChargingStatusResponse
	require.NoError(t, json.Unmarshal([]byte(sampleStatusIdle), &res))
	require.NoError(t, res.Err())

	data := res.BikeChargingData
	assert.False(t, data.ChargingStatus)
	assert.False(t, data.PluggedIn)
	assert.Equal(t, 65.0, data.BatteryPercentage)
	assert.Equal(t, 67.0, data.Range)
	assert.Equal(t, 0.0, float64(data.TimeToMaxLimit))
	assert.Equal(t, int64(80), data.MaxLimit)
	assert.InDelta(t, 550.63, data.Odometer, 0.01)
	assert.Equal(t, 4.0, float64(data.DurationElapsed))
	assert.Equal(t, "seconds", data.DurationUnit)
}

func TestChargingStatusAge(t *testing.T) {
	for _, tc := range []struct {
		elapsed, unit string
		want          time.Duration
	}{
		{"4", "seconds", 4 * time.Second},
		{"2", "minutes", 2 * time.Minute},
		{"1.5", "hours", 90 * time.Minute},
		{"0", "", 0},
	} {
		s := ChargingStatus{DurationUnit: tc.unit}
		require.NoError(t, json.Unmarshal([]byte(`"`+tc.elapsed+`"`), &s.DurationElapsed))
		assert.Equal(t, tc.want, s.Age(), tc.elapsed+" "+tc.unit)
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
	var bikes BikesResponse
	require.NoError(t, json.Unmarshal([]byte(sampleBikes), &bikes))
	require.Len(t, bikes.Bikes, 1)
	assert.Equal(t, "100000001", bikes.Bikes[0].ID)
	assert.Equal(t, "7TM3GDYD6SB000000", bikes.Bikes[0].VIN)
	assert.Equal(t, "S2 Mulholland", bikes.Bikes[0].Model)
	assert.False(t, bikes.Bikes[0].PairingStatus)

	var paired BikesResponse
	require.NoError(t, json.Unmarshal([]byte(samplePairStatusAfter), &paired))
	require.Len(t, paired.Bikes, 1)
	assert.True(t, paired.Bikes[0].PairingStatus)
}

func TestLocationResponse(t *testing.T) {
	var res LocationResponse
	require.NoError(t, json.Unmarshal([]byte(sampleLocationIdle), &res))
	require.NoError(t, res.Err())
	assert.Equal(t, 48.1371, res.Data.Latitude)
	assert.Equal(t, 11.5754, res.Data.Longitude)
	assert.Equal(t, 3, res.Data.FixType)
}

func TestSessionResponse(t *testing.T) {
	var res SessionResponse
	require.NoError(t, json.Unmarshal([]byte(sampleSession), &res))
	require.NoError(t, res.Err())
	assert.NotEmpty(t, res.JWT)
	assert.True(t, res.TermsAccepted)
	assert.True(t, tokenExpiry(res.JWT).IsZero(), "live tokens carry no exp claim")
}
