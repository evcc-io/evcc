package meter

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestBatteryControlState(t *testing.T) {
	raw := `{"Mode":"automatic","NativeMode":"10","Power":-200,"Soc":95,"NativeCharging":true,"Sample":"one","LeaseID":"cannot-inject-ownership"}`
	r := &batteryControlStateReader{get: func() (string, error) { return raw, nil }}
	state, err := r.BatteryControlState()
	require.NoError(t, err)
	require.Equal(t, api.BatteryOperatingAuto, state.Mode)
	require.Equal(t, "10", state.NativeMode)
	require.Equal(t, -200.0, state.Power)
	require.True(t, state.NativeCharging)
	require.Empty(t, state.LeaseID)
	// Repeated responses do not refresh the age of a frozen device snapshot.
	r.observed = time.Now().Add(-time.Minute)
	repeated, err := r.BatteryControlState()
	require.NoError(t, err)
	require.Equal(t, r.observed, repeated.ObservedAt)
	raw = `{"Mode":"manual","NativeMode":"1","Power":0,"Soc":50,"NativeCharging":false,"Sample":"two"}`
	changed, err := r.BatteryControlState()
	require.NoError(t, err)
	require.Equal(t, api.BatteryOperatingManual, changed.Mode)
	require.True(t, changed.ObservedAt.After(repeated.ObservedAt))
}

func TestBatteryControlStateRejectsIncomplete(t *testing.T) {
	for _, raw := range []string{
		`{}`, `invalid`,
		`{"Power":0,"Soc":50,"Sample":"one"}`,
		`{"Power":0,"Soc":101,"NativeCharging":false,"Sample":"one"}`,
		`{"Power":0,"Soc":50,"NativeCharging":false}`,
	} {
		t.Run(raw, func(t *testing.T) {
			r := &batteryControlStateReader{get: func() (string, error) { return raw, nil }}
			_, err := r.BatteryControlState()
			require.Error(t, err)
		})
	}
	r := &batteryControlStateReader{get: func() (string, error) { return "", errors.New("offline") }}
	_, err := r.BatteryControlState()
	require.ErrorContains(t, err, "offline")
	r.get = func() (string, error) {
		return `{"Mode":"unexpected","Power":0,"Soc":50,"NativeCharging":false,"Sample":"one"}`, nil
	}
	state, err := r.BatteryControlState()
	require.NoError(t, err)
	require.Equal(t, api.BatteryOperatingUnknown, state.Mode)
}
