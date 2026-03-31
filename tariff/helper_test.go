package tariff

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeRatesAfter(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	old := api.Rates{rate(1, 1), rate(2, 2)}
	new := api.Rates{rate(2, 2), rate(3, 3)}
	combined := api.Rates{rate(1, 1), rate(2, 2), rate(3, 3)}

	data := util.NewMonitor[api.Rates](time.Hour).WithClock(clock)

	data.Set(old)
	res, err := data.Get()
	require.NoError(t, err)
	assert.Equal(t, old, res)

	for i, tc := range []struct {
		new, expected api.Rates
		ts            time.Time
	}{
		{new, combined, clock.Now()},
		{new, combined, clock.Now().Add(SlotDuration)},
		{nil, combined, clock.Now()},
		{nil, combined, clock.Now().Add(SlotDuration)},
		{new, combined, clock.Now().Add(time.Hour)},
		{new, combined, clock.Now().Add(time.Hour + SlotDuration)},
		{new, combined, now.With(clock.Now().Add(time.Hour + 30*time.Minute)).BeginningOfHour()},
		{new, combined, now.With(clock.Now().Add(time.Hour + 30*time.Minute)).BeginningOfHour().Add(SlotDuration)},
		{new, new, clock.Now().Add(2 * time.Hour)},
		{new, new, clock.Now().Add(2*time.Hour + SlotDuration)},
	} {
		t.Logf("%d. %+v", i+1, tc)

		mergeRatesAfter(data, tc.new, tc.ts)

		res, err := data.Get()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, res)
	}
}

type runner struct {
	res error
}

func (r *runner) run(done chan error) {
	if r.res == nil {
		close(done)
	} else {
		done <- r.res
	}
}

func TestRunOrQError(t *testing.T) {
	{
		res, err := runOrError(&runner{nil})
		require.NoError(t, err)
		require.NotNil(t, res)
	}
	{
		res, err := runOrError(&runner{errors.New("foo")})
		require.Error(t, err)
		require.Nil(t, res)
	}
}

func statusError(code int) *request.StatusError {
	rec := httptest.NewRecorder()
	rec.Code = code
	resp := rec.Result()
	resp.Request = &http.Request{Method: http.MethodGet, URL: &url.URL{}}
	return request.NewStatusError(resp)
}

func TestBackoffPermanentError(t *testing.T) {
	tt := []struct {
		name      string
		code      int
		permanent bool
	}{
		{name: "400 Bad Request is permanent", code: http.StatusBadRequest, permanent: true},
		{name: "401 Unauthorized is permanent", code: http.StatusUnauthorized, permanent: true},
		{name: "404 Not Found is permanent", code: http.StatusNotFound, permanent: true},
		{name: "500 Internal Server Error is permanent", code: http.StatusInternalServerError, permanent: true},
		{name: "429 Too Many Requests is NOT permanent (transient)", code: http.StatusTooManyRequests, permanent: false},
		{name: "503 Service Unavailable is NOT permanent (transient)", code: http.StatusServiceUnavailable, permanent: false},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := backoffPermanentError(statusError(tc.code))
			var pe *backoff.PermanentError
			isPermanent := errors.As(err, &pe)
			if tc.permanent {
				assert.True(t, isPermanent, "expected permanent error for HTTP %d", tc.code)
			} else {
				assert.False(t, isPermanent, "expected non-permanent error for HTTP %d", tc.code)
			}
		})
	}
}
