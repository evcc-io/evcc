package charger

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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

// a vehicle woken from sleep may still report the pre-sleep charge state, which
// must not be taken for a disconnect while the grace window is open (#33323)
func TestStatusAfterWakeup(t *testing.T) {
	ctrl := gomock.NewController(t)

	type vehicle struct {
		*api.MockVehicle
		*api.MockChargeState
	}
	v := &vehicle{api.NewMockVehicle(ctrl), api.NewMockChargeState(ctrl)}

	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetVehicle().Return(v).AnyTimes()

	c := &VehicleApi{lp: lp}

	// vehicle asleep: connected, so the loadpoint can wake it
	v.MockChargeState.EXPECT().Status().Return(api.StatusNone, api.ErrAsleep)
	status, err := c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// answers again but still reports disconnected: not believed yet
	v.MockChargeState.EXPECT().Status().Return(api.StatusA, nil)
	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)

	// grace expired: disconnect is real
	c.asleepAt = time.Now().Add(-asleepGrace - time.Second)
	v.MockChargeState.EXPECT().Status().Return(api.StatusA, nil)
	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)
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
