package nissan

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// batteryFunc returns a StatusResponse with the given Updated time and
// records how many times it was called.
func batteryFunc(updated time.Time) (func() (StatusResponse, error), *int) {
	calls := 0
	return func() (StatusResponse, error) {
		calls++
		res := StatusResponse{}
		res.Attributes.Updated = updated
		return res, nil
	}, &calls
}

// refreshFunc records calls and returns nil.
func refreshFunc() (func() (ActionResponse, error), *int) {
	calls := 0
	return func() (ActionResponse, error) {
		calls++
		return ActionResponse{}, nil
	}, &calls
}

// TestStatusV2SynthesizedNowSkipsRefresh is the core regression guard.
//
// When the Nissan Ariya (v2) API returns a timestamp 30 min–7 h in the past,
// BatteryStatus synthesizes Updated = time.Now(). Provider.status must treat
// that as fresh data (time.Since < 5 min expiry) and return the result
// immediately without calling the refresh endpoint.
func TestStatusV2SynthesizedNowSkipsRefresh(t *testing.T) {
	p := &Provider{expiry: 5 * time.Minute}

	battery, batteryCalls := batteryFunc(time.Now())
	refresh, refreshCalls := refreshFunc()

	res, err := p.status(battery, refresh)
	require.NoError(t, err)
	assert.Equal(t, 1, *batteryCalls, "battery should be called once")
	assert.Equal(t, 0, *refreshCalls, "refresh must not be called when data is fresh")
	assert.Equal(t, 0, res.Attributes.BatteryLevel)
}

// TestStatusZeroTimestampSkipsRefresh verifies that a zero Updated (no
// timestamp in the API response at all) is treated as valid, matching the
// `updated.IsZero()` branch in Provider.status.
func TestStatusZeroTimestampSkipsRefresh(t *testing.T) {
	p := &Provider{expiry: 5 * time.Minute}

	battery, batteryCalls := batteryFunc(time.Time{})
	refresh, refreshCalls := refreshFunc()

	_, err := p.status(battery, refresh)
	require.NoError(t, err)
	assert.Equal(t, 1, *batteryCalls)
	assert.Equal(t, 0, *refreshCalls, "refresh must not be called when Updated is zero")
}

// TestStatusRawPastTimestampTriggersRefresh documents the bug that the
// timestamp fix resolved: if Updated held the raw Kamereon v2 timestamp
// (30 min–7 h old), time.Since(updated) > 5 min expiry, so every poll would
// trigger a refresh request.
func TestStatusRawPastTimestampTriggersRefresh(t *testing.T) {
	p := &Provider{expiry: 5 * time.Minute}

	// Simulate what the old Updated() method returned for v2: the raw API timestamp.
	past2h := time.Now().Add(-2 * time.Hour)
	battery, batteryCalls := batteryFunc(past2h)
	refresh, refreshCalls := refreshFunc()

	_, err := p.status(battery, refresh)
	require.ErrorIs(t, err, api.ErrMustRetry,
		"stale Updated (2 h old) should trigger a refresh and return ErrMustRetry")
	assert.Equal(t, 1, *batteryCalls)
	assert.Equal(t, 1, *refreshCalls, "refresh must be called when Updated is beyond expiry")
}
