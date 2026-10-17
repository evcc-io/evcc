package fiat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const notPluggedIn = "Deep refresh not allowed if the vehicle is not plugged-in"

var errForbidden = func() error {
	req, _ := http.NewRequest(http.MethodPost, "https://example.invalid", nil)
	return request.NewStatusError(&http.Response{StatusCode: http.StatusForbidden, Request: req})
}()

func testStatus(t *testing.T, age time.Duration, pluggedIn bool, charging, schedules string) StatusResponse {
	t.Helper()

	var res StatusResponse
	js := fmt.Sprintf(`{"timestamp":%d,"evInfo":{"battery":{"plugInStatus":%t,"chargingStatus":%q,"stateOfCharge":71},"schedules":%s}}`,
		time.Now().Add(-age).UnixMilli(), pluggedIn, charging, schedules)
	require.NoError(t, json.Unmarshal([]byte(js), &res))

	return res
}

func action(res ActionResponse, err error) func() (ActionResponse, error) {
	return func() (ActionResponse, error) { return res, err }
}

func TestStatusDeepRefresh(t *testing.T) {
	for _, tc := range []struct {
		name      string
		age       time.Duration
		pluggedIn bool
		started   time.Duration // refresh started this long ago
		action    func() (ActionResponse, error)
		refresh   bool // deep refresh requested
		pending   bool // waiting for refresh afterwards
		unplugged bool
		err       error
	}{
		{name: "fresh status", age: time.Minute},
		{name: "expired, not plugged in", age: time.Hour, unplugged: true},
		{name: "expired, plugged in", age: time.Hour, pluggedIn: true, action: action(ActionResponse{ResponseStatus: "pending"}, nil), refresh: true, pending: true, err: api.ErrMustRetry},
		{name: "rejected, not plugged in", age: time.Hour, pluggedIn: true, action: action(ActionResponse{DebugMsg: notPluggedIn}, errForbidden), refresh: true, unplugged: true},
		{name: "rejected, other reason", age: time.Hour, pluggedIn: true, action: action(ActionResponse{DebugMsg: "Remote command not authorized"}, errForbidden), refresh: true, err: errForbidden},
		{name: "rejected pin authentication", age: time.Hour, pluggedIn: true, action: action(ActionResponse{}, errForbidden), refresh: true, err: errForbidden},
		{name: "unplugged while refreshing", age: time.Hour, started: time.Second, unplugged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var refreshed bool

			v := &Provider{
				expiry: 5 * time.Minute,
				action: func(action, cmd string) (ActionResponse, error) {
					refreshed = true
					require.NotNil(t, tc.action, "unexpected deep refresh")
					return tc.action()
				},
			}

			if tc.started > 0 {
				v.refreshTime = time.Now().Add(-tc.started)
			}

			res, err := v.status(func() (StatusResponse, error) {
				return testStatus(t, tc.age, tc.pluggedIn, "NOT_CHARGING", "[]"), nil
			})

			assert.Equal(t, tc.refresh, refreshed, "deep refresh")
			assert.Equal(t, tc.pending, !v.refreshTime.IsZero(), "pending refresh")
			assert.Equal(t, tc.unplugged, res.unplugged, "unplugged")
			assert.Equal(t, 71.0, res.EvInfo.Battery.StateOfCharge)

			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStatusRejectedRefresh(t *testing.T) {
	v := &Provider{
		expiry: 5 * time.Minute,
		action: func(string, string) (ActionResponse, error) {
			return ActionResponse{DebugMsg: notPluggedIn}, errForbidden
		},
	}

	// outdated status still claims charging
	v.statusG = func() (StatusResponse, error) {
		return v.status(func() (StatusResponse, error) {
			return testStatus(t, time.Hour, true, "CHARGING", "[]"), nil
		})
	}

	status, err := v.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusA, status)
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestControllerUnplugged(t *testing.T) {
	var paths []string

	fiat := &API{
		identity: &Identity{uid: "uid", creds: &credentials{
			AccessKeyId: "key", SecretKey: "secret", SessionToken: "token", Expiration: float64(time.Now().Add(time.Hour).Unix()),
		}},
		Helper: &request.Helper{Client: &http.Client{Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
			paths = append(paths, req.URL.Path)
			body := `{"responseStatus":"pending"}`
			if strings.HasSuffix(req.URL.Path, "/ignite/pin/authenticate") {
				body = `{"token":"token"}`
			}
			return &http.Response{StatusCode: http.StatusOK, Request: req, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		})}},
	}

	// outdated status with an enabled charge schedule while not plugged in
	v := &Provider{expiry: 5 * time.Minute}
	v.statusG = func() (StatusResponse, error) {
		return v.status(func() (StatusResponse, error) {
			return testStatus(t, time.Hour, false, "NOT_CHARGING", `[{"enableScheduleType":true,"scheduleType":"CHARGE"},{},{}]`), nil
		})
	}

	c := NewController(v, fiat, util.NewLogger("fiat-test"), "vin", "1234")

	// outdated schedules are not written back
	assert.ErrorIs(t, c.ChargeEnable(true), api.ErrMustRetry)
	assert.Empty(t, paths)

	// outdated schedules do not suppress the wake up
	require.NoError(t, c.WakeUp())
	require.NotEmpty(t, paths)
	assert.True(t, strings.HasSuffix(paths[len(paths)-1], "/ev/chargenow"))
}
