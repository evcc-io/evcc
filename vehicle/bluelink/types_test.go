package bluelink

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVehicleStatusRange(t *testing.T) {
	tc := []struct {
		name string
		json string
		want int64
		err  error
	}{
		{
			// real payload from issue #29728: UK Hyundai with miles unit
			name: "miles (unit=3)",
			json: `{"evStatus":{"drvDistance":[{"rangeByFuel":{"evModeRange":{"value":168.38834951456312,"unit":3}}}]}}`,
			want: 271, // 168.38834951456312 * 1.60934 ≈ 271.04
		},
		{
			name: "kilometers (unit=1)",
			json: `{"evStatus":{"drvDistance":[{"rangeByFuel":{"evModeRange":{"value":271,"unit":1}}}]}}`,
			want: 271,
		},
		{
			name: "no evStatus",
			json: `{}`,
			err:  api.ErrNotAvailable,
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			var s VehicleStatus
			require.NoError(t, json.Unmarshal([]byte(c.json), &s))
			got, err := s.Range()
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestStatusLatestResponseOdometer(t *testing.T) {
	tc := []struct {
		name string
		json string
		want float64
		err  error
	}{
		{
			// real payload from issue #29728: km unit, returned as-is
			name: "kilometers (unit=1)",
			json: `{"resMsg":{"vehicleStatusInfo":{"odometer":{"value":88038.5,"unit":1}}}}`,
			want: 88038.5,
		},
		{
			name: "miles (unit=3)",
			json: `{"resMsg":{"vehicleStatusInfo":{"odometer":{"value":1000,"unit":3}}}}`,
			want: 1609.34,
		},
		{
			name: "no odometer",
			json: `{}`,
			err:  api.ErrNotAvailable,
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			var r StatusLatestResponse
			require.NoError(t, json.Unmarshal([]byte(c.json), &r))
			got, err := r.Odometer()
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, c.want, got, 0.001)
		})
	}
}
