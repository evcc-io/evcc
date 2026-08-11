package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForecastRates(t *testing.T) {
	start := time.Unix(1735689600, 0)

	for _, tc := range []struct {
		desc  string
		rates api.Rates
		want  string
	}{
		{desc: "nil", rates: nil, want: "null"},
		{desc: "empty", rates: api.Rates{}, want: "null"},
		{
			desc: "slots",
			rates: api.Rates{
				{Start: start, End: start.Add(time.Hour), Value: 0.25},
				{Start: start.Add(time.Hour), End: start.Add(2 * time.Hour), Value: -0.1},
			},
			want: "[[1735689600,1735693200,0.25],[1735693200,1735696800,-0.1]]",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			b, err := json.Marshal(forecastRates(tc.rates))
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(b))
		})
	}
}

func TestTimeseriesMarshal(t *testing.T) {
	start := time.Unix(1735689600, 0)

	for _, tc := range []struct {
		desc string
		ts   timeseries
		want string
	}{
		{desc: "nil", ts: nil, want: "null"},
		{desc: "empty", ts: timeseries{}, want: "[]"},
		{
			desc: "entries",
			ts: timeseries{
				{Timestamp: start, Value: 1000},
				{Timestamp: start.Add(time.Hour), Value: 0},
			},
			want: "[[1735689600,1000],[1735693200,0]]",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			b, err := json.Marshal(tc.ts)
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(b))

			b, err = tc.ts.MarshalBytes()
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(b))
		})
	}
}
