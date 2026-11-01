package bluelink

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statusErr(code int) error {
	req, _ := http.NewRequest(http.MethodPost, "https://example.invalid/control/charge", nil)
	return request.NewStatusError(&http.Response{StatusCode: code, Request: req})
}

func TestChargeControlShouldBackoff(t *testing.T) {
	assert.False(t, chargeControlShouldBackoff(nil))
	assert.True(t, chargeControlShouldBackoff(statusErr(http.StatusForbidden)))
	assert.True(t, chargeControlShouldBackoff(statusErr(http.StatusBadRequest)))
	assert.False(t, chargeControlShouldBackoff(statusErr(http.StatusUnauthorized)))
	assert.False(t, chargeControlShouldBackoff(statusErr(http.StatusBadGateway)))
	assert.True(t, chargeControlShouldBackoff(errors.Join(
		statusErr(http.StatusForbidden),
		statusErr(http.StatusBadRequest),
	)))
}

func TestEnableChargeBackoffOnForbidden(t *testing.T) {
	var calls int
	p := &Provider{
		chargeS: func(bool) error {
			calls++
			return statusErr(http.StatusForbidden)
		},
		chargeBackoff: time.Minute,
	}

	err := p.EnableCharge(false)
	require.Error(t, err)
	assert.Equal(t, 1, calls)

	err = p.EnableCharge(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "charge control paused")
	assert.Equal(t, 1, calls, "403 must not be retried until backoff expires")
}

func TestEnableChargeDoesNotBackoffTransientErrors(t *testing.T) {
	var calls int
	p := &Provider{
		chargeS: func(bool) error {
			calls++
			return statusErr(http.StatusBadGateway)
		},
		chargeBackoff: time.Minute,
	}

	require.Error(t, p.EnableCharge(false))
	require.Error(t, p.EnableCharge(false))
	assert.Equal(t, 2, calls)
}

func TestEnableChargeClearsBackoffOnSuccess(t *testing.T) {
	var calls int
	p := &Provider{
		chargeS: func(bool) error {
			calls++
			if calls == 1 {
				return statusErr(http.StatusForbidden)
			}
			return nil
		},
		chargeBackoff: time.Minute,
	}

	require.Error(t, p.EnableCharge(false))
	p.chargeUntil = time.Time{}
	require.NoError(t, p.EnableCharge(true))
	require.NoError(t, p.EnableCharge(true))
	assert.Equal(t, 3, calls)
}
