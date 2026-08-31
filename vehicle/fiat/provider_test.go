package fiat

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statusErr builds a request.StatusError carrying the given HTTP status code and
// request path.
func statusErr(code int, path string) error {
	req := &http.Request{Method: http.MethodPost, URL: &url.URL{Scheme: "https", Host: "example.invalid", Path: path}}
	return request.NewStatusError(&http.Response{StatusCode: code, Request: req})
}

const (
	pathDeepRefresh = "/v1/accounts/1/vehicles/1/ev"
	pathPinAuth     = "/v1/accounts/1/ignite/pin/authenticate"
)

// staleStatus returns a status response older than the provider expiry.
func staleStatus() StatusResponse {
	var res StatusResponse
	res.Timestamp = TimeMillis{time.Now().Add(-10 * time.Minute)}
	res.VehicleInfo.Odometer.Odometer.Value = 31650
	return res
}

func TestProviderStatus(t *testing.T) {
	t.Run("fresh data is returned without refresh", func(t *testing.T) {
		var res StatusResponse
		res.Timestamp = TimeMillis{time.Now()}
		res.VehicleInfo.Odometer.Odometer.Value = 31650

		called := false
		v := &Provider{
			expiry: 5 * time.Minute,
			action: func(action, cmd string) (ActionResponse, error) {
				called = true
				return ActionResponse{}, nil
			},
		}

		got, err := v.status(func() (StatusResponse, error) { return res, nil })
		require.NoError(t, err)
		assert.False(t, called, "no refresh expected for fresh data")
		assert.Equal(t, 31650, got.VehicleInfo.Odometer.Odometer.Value)
	})

	t.Run("refused refresh returns last known data without error", func(t *testing.T) {
		// vehicle refuses DEEPREFRESH with 403 when it is not plugged in
		calls := 0
		v := &Provider{
			expiry: 5 * time.Minute,
			action: func(action, cmd string) (ActionResponse, error) {
				calls++
				return ActionResponse{}, statusErr(http.StatusForbidden, pathDeepRefresh)
			},
		}

		got, err := v.status(func() (StatusResponse, error) { return staleStatus(), nil })
		require.NoError(t, err, "a refused (not plugged in) refresh must not surface as error")
		assert.Equal(t, 1, calls, "deep refresh should be attempted once")
		assert.Equal(t, 31650, got.VehicleInfo.Odometer.Odometer.Value, "last known data must be preserved")
		assert.True(t, v.refreshTime.IsZero(), "no retry loop should be started for a refused refresh")
	})

	t.Run("pin authentication failure is surfaced", func(t *testing.T) {
		// a 403 from the pin authentication step must not be masked as a refusal
		v := &Provider{
			expiry: 5 * time.Minute,
			action: func(action, cmd string) (ActionResponse, error) {
				return ActionResponse{}, statusErr(http.StatusForbidden, pathPinAuth)
			},
		}

		_, err := v.status(func() (StatusResponse, error) { return staleStatus(), nil })
		require.Error(t, err, "a pin authentication failure must surface as error")
		assert.NotErrorIs(t, err, api.ErrMustRetry)
	})

	t.Run("accepted refresh arms the retry window", func(t *testing.T) {
		v := &Provider{
			expiry: 5 * time.Minute,
			action: func(action, cmd string) (ActionResponse, error) {
				return ActionResponse{ResponseStatus: "pending"}, nil
			},
		}

		_, err := v.status(func() (StatusResponse, error) { return staleStatus(), nil })
		assert.ErrorIs(t, err, api.ErrMustRetry)
		assert.False(t, v.refreshTime.IsZero(), "retry window should be armed after an accepted refresh")
	})

	t.Run("unexpected refresh error is surfaced", func(t *testing.T) {
		v := &Provider{
			expiry: 5 * time.Minute,
			action: func(action, cmd string) (ActionResponse, error) {
				return ActionResponse{}, statusErr(http.StatusInternalServerError, pathDeepRefresh)
			},
		}

		_, err := v.status(func() (StatusResponse, error) { return staleStatus(), nil })
		require.Error(t, err)
		assert.NotErrorIs(t, err, api.ErrMustRetry)
	})
}
