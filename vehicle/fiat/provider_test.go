package fiat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statusResponse creates a status response uploaded by the vehicle the given time ago
func statusResponse(t *testing.T, age time.Duration, pluggedIn bool) StatusResponse {
	t.Helper()

	var res StatusResponse
	js := fmt.Sprintf(`{"timestamp":%d,"evInfo":{"battery":{"plugInStatus":%t,"stateOfCharge":71}}}`, time.Now().Add(-age).UnixMilli(), pluggedIn)
	require.NoError(t, json.Unmarshal([]byte(js), &res))

	return res
}

// forbidden simulates an HTTP 403 response, the name is only set by the deep refresh endpoint
func forbidden(name string) func() (ActionResponse, error) {
	return func() (ActionResponse, error) {
		resp := &http.Response{
			StatusCode: http.StatusForbidden,
			Request:    &http.Request{Method: http.MethodPost, URL: &url.URL{Path: "/ev"}},
		}
		return ActionResponse{Name: name}, request.NewStatusError(resp)
	}
}

func TestStatusDeepRefresh(t *testing.T) {
	pending := func() (ActionResponse, error) {
		return ActionResponse{ResponseStatus: "pending"}, nil
	}

	for _, tc := range []struct {
		name      string
		age       time.Duration
		pluggedIn bool
		action    func() (ActionResponse, error)
		refresh   bool
		err       error
	}{
		{"fresh status", time.Minute, false, nil, false, nil},
		{"expired, not plugged in", time.Hour, false, nil, false, nil},
		{"expired, plugged in", time.Hour, true, pending, true, api.ErrMustRetry},
		{"expired, deep refresh rejected", time.Hour, true, forbidden("FORBIDDEN"), true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var refreshed bool

			v := &Provider{
				expiry: 5 * time.Minute,
				action: func(action, cmd string) (ActionResponse, error) {
					refreshed = true
					assert.Equal(t, "ev", action)
					assert.Equal(t, "DEEPREFRESH", cmd)

					if tc.action == nil {
						return ActionResponse{}, errors.New("unexpected deep refresh")
					}
					return tc.action()
				},
			}

			res, err := v.status(func() (StatusResponse, error) {
				return statusResponse(t, tc.age, tc.pluggedIn), nil
			})

			assert.Equal(t, tc.refresh, refreshed, "deep refresh")
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}

			// last known values are always returned
			require.NotNil(t, res.EvInfo)
			assert.Equal(t, 71.0, res.EvInfo.Battery.StateOfCharge)
		})
	}
}

func TestStatusPinAuthForbidden(t *testing.T) {
	v := &Provider{
		expiry: 5 * time.Minute,
		action: func(action, cmd string) (ActionResponse, error) {
			// pin authentication fails before the deep refresh endpoint is called
			return forbidden("")()
		},
	}

	_, err := v.status(func() (StatusResponse, error) {
		return statusResponse(t, time.Hour, true), nil
	})

	se, ok := errors.AsType[*request.StatusError](err)
	require.True(t, ok, "expected status error, got %v", err)
	assert.Equal(t, http.StatusForbidden, se.StatusCode())
}
