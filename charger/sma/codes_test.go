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
		{StatusNotConnected, api.StatusA, false},
		{StatusSleeping, api.StatusB, false},
		{StatusFinished, api.StatusB, false},
		{StatusLocked, api.StatusB, false},
		{StatusCharging, api.StatusC, false},
		{StatusWarning, api.StatusNone, true},
		{StatusError, api.StatusNone, true},
	} {
		status, err := ChargeStatus(tc.code)
		assert.Equal(t, tc.status, status, tc.code)
		assert.Equal(t, tc.err, err != nil, tc.code)
	}
}

func TestStatusReason(t *testing.T) {
	assert.Equal(t, api.ReasonWaitingForAuthorization, StatusReason(StatusLocked, true))
	assert.Equal(t, api.ReasonUnknown, StatusReason(StatusLocked, false))
	assert.Equal(t, api.ReasonUnknown, StatusReason(StatusSleeping, true))
}

func TestParseCode(t *testing.T) {
	c, err := ParseCode("5169")
	require.NoError(t, err)
	assert.Equal(t, StatusLocked, c)
	assert.Equal(t, "5169", c.String())

	_, err = ParseCode("invalid")
	assert.Error(t, err)
}
