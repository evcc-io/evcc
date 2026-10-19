package charger

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// statusErr builds the error a plugin returns for the given response
func statusErr(t *testing.T, code int, body string) error {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "http://vehicle.test", nil)
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}

	_, err = request.ReadBody(resp)
	require.Error(t, err)

	return err
}

func TestAsleep(t *testing.T) {
	sleeping := `{"response":{"result":false,"reason":"vehicle is sleeping"}}`

	// TeslaBleHttpProxy sleeping response
	require.ErrorIs(t, asleep(statusErr(t, http.StatusServiceUnavailable, sleeping)), api.ErrAsleep)

	// 503 for any other reason is not asleep
	other := statusErr(t, http.StatusServiceUnavailable, `{"response":{"reason":"vehicle is offline"}}`)
	require.NotErrorIs(t, asleep(other), api.ErrAsleep)

	// non-503 with sleeping body is not asleep
	require.NotErrorIs(t, asleep(statusErr(t, http.StatusInternalServerError, sleeping)), api.ErrAsleep)

	// non-json body and nil error pass through
	require.NotErrorIs(t, asleep(statusErr(t, http.StatusServiceUnavailable, "sleeping")), api.ErrAsleep)
	require.NoError(t, asleep(nil))
}

func TestAsleepTeslaMate(t *testing.T) {
	unavailable := `{"error":"vehicle unavailable: {error: \"vehicle unavailable:\"}","error_description":"","response":null}`
	require.ErrorIs(t, asleep(statusErr(t, http.StatusRequestTimeout, unavailable)), api.ErrAsleep)

	for _, err := range []error{
		statusErr(t, http.StatusServiceUnavailable, unavailable),
		statusErr(t, http.StatusRequestTimeout, `{"error":"request timed out"}`),
		statusErr(t, http.StatusRequestTimeout, `{"response":{"reason":"vehicle is sleeping"}}`),
		statusErr(t, http.StatusRequestTimeout, "vehicle unavailable"),
	} {
		require.Same(t, err, asleep(err))
	}
}

// asleepVehicle is a vehicle whose charge state can be queried
type asleepVehicle struct {
	*api.MockVehicle
	*api.MockChargeState
}

func TestVehicleApiStatusAsleep(t *testing.T) {
	ctrl := gomock.NewController(t)
	lp := loadpoint.NewMockAPI(ctrl)
	cs := api.NewMockChargeState(ctrl)
	lp.EXPECT().GetVehicle().Return(asleepVehicle{api.NewMockVehicle(ctrl), cs}).AnyTimes()

	c := &VehicleApi{lp: lp}

	// unknown last status: wake vehicle
	cs.EXPECT().Status().Return(api.StatusNone, api.ErrAsleep)
	status, err := c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// awake and disconnected
	cs.EXPECT().Status().Return(api.StatusA, nil)
	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)

	// asleep after disconnected: stay disconnected, don't wake
	cs.EXPECT().Status().Return(api.StatusNone, api.ErrAsleep)
	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)

	// awake and connected
	cs.EXPECT().Status().Return(api.StatusB, nil)
	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// asleep after connected: wake vehicle
	cs.EXPECT().Status().Return(api.StatusNone, api.ErrAsleep)
	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)
}
