package sma

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChargeStatus(t *testing.T) {
	for _, tc := range []struct {
		code   Code
		status api.ChargeStatus
		err    bool
	}{
		{NotConnected, api.StatusA, false},
		{SleepMode, api.StatusB, false},
		{ChargeTargetReached, api.StatusB, false},
		{StationLocked, api.StatusB, false},
		{ActiveMode, api.StatusC, false},
		{Warning, api.StatusNone, true},
		{StationFault, api.StatusNone, true},
	} {
		status, err := ChargeStatus(tc.code)
		assert.Equal(t, tc.status, status, tc.code)
		assert.Equal(t, tc.err, err != nil, tc.code)
	}
}

func TestStatusReason(t *testing.T) {
	assert.Equal(t, api.ReasonWaitingForAuthorization, StatusReason(Yes, ChargingLock))
	assert.Equal(t, api.ReasonUnknown, StatusReason(Yes, ChargingRelease))
	assert.Equal(t, api.ReasonUnknown, StatusReason(No, ChargingLock))
}

func TestParseCode(t *testing.T) {
	c, err := ParseCode("5169")
	require.NoError(t, err)
	assert.Equal(t, StationLocked, c)
	assert.Equal(t, "5169", c.String())

	_, err = ParseCode("invalid")
	assert.Error(t, err)
}
