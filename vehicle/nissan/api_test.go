package nissan

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newTestAPI(body string) *API {
	v := &API{Helper: request.NewHelper(util.NewLogger("test"))}
	v.Client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": {"application/json"}},
		}, nil
	})
	return v
}

// TestBatteryStatusV2IgnoresPastTimestamp verifies that BatteryStatus for v2
// ignores the raw timestamp value and sets Updated to roughly now, regardless
// of how old the reported timestamp is.
//
// Background: the Nissan Ariya (Kamereon v2) returns timestamps that are
// 30 min–7 h in the past. Using that raw value as Updated caused Provider.status
// to always consider the result stale (time.Since > 5 min expiry) and issue
// an endless stream of refresh requests. Setting Updated = time.Now() on a
// non-nil timestamp breaks that loop.
func TestBatteryStatusV2IgnoresPastTimestamp(t *testing.T) {
	past30m := time.Now().Add(-30 * time.Minute)
	past2h := time.Now().Add(-2 * time.Hour)
	past7h := time.Now().Add(-7 * time.Hour)

	cases := []struct {
		name      string
		timestamp time.Time
	}{
		{"30 minutes old", past30m},
		{"2 hours old (mid-range)", past2h},
		{"7 hours old (worst case)", past7h},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"batteryLevel":80,"chargeStatus":1,"timestamp":"` + tc.timestamp.UTC().Format(time.RFC3339) + `"}`
			res, err := newTestAPI(body).BatteryStatus("VIN", "v2")
			require.NoError(t, err)

			// Updated must be close to now, not to the stale API timestamp.
			assert.WithinDuration(t, time.Now(), res.Attributes.Updated, time.Minute,
				"v2 Updated should be synthesized as now")
			assert.False(t, res.Attributes.Updated.Equal(tc.timestamp),
				"v2 must not use the raw past timestamp as Updated")
		})
	}
}

// TestBatteryStatusV2NoTimestampYieldsZero checks that a v2 response without
// a timestamp field leaves Updated as zero (no timestamp available at all).
func TestBatteryStatusV2NoTimestampYieldsZero(t *testing.T) {
	body := `{"batteryLevel":80,"chargeStatus":1}`
	res, err := newTestAPI(body).BatteryStatus("VIN", "v2")
	require.NoError(t, err)
	assert.True(t, res.Attributes.Updated.IsZero(), "v2 without timestamp should leave Updated zero")
}

// TestBatteryStatusV1HonorsPastTimestamp contrasts v2 behaviour: v1 uses
// lastUpdateTime directly, so a past timestamp is preserved in Updated.
func TestBatteryStatusV1HonorsPastTimestamp(t *testing.T) {
	past2h := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	body := `{"batteryLevel":80,"chargeStatus":1,"lastUpdateTime":"` + past2h.Format(timeFormat) + `"}`
	res, err := newTestAPI(body).BatteryStatus("VIN", "v1")
	require.NoError(t, err)
	assert.Equal(t, past2h, res.Attributes.Updated, "v1 should use the reported lastUpdateTime verbatim")
}

// TestBatteryStatusV1NoTimestampYieldsZero checks that a v1 response without
// lastUpdateTime leaves Updated as zero.
func TestBatteryStatusV1NoTimestampYieldsZero(t *testing.T) {
	body := `{"batteryLevel":80,"chargeStatus":1}`
	res, err := newTestAPI(body).BatteryStatus("VIN", "v1")
	require.NoError(t, err)
	assert.True(t, res.Attributes.Updated.IsZero(), "v1 without lastUpdateTime should leave Updated zero")
}
