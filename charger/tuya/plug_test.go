package tuya

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugValue(t *testing.T) {
	var dps map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{"1": true, "17": 43, "18": 1057, "19": 1579, "20": 2392}`), &dps))

	for _, tc := range []struct {
		dp    string
		scale float64
		want  float64
	}{
		{plugDpPower, 10, 157.9},
		{plugDpCurrent, 1e3, 1.057},
		{plugDpVoltage, 10, 239.2},
	} {
		res, err := plugValue(dps, tc.dp, tc.scale)
		require.NoError(t, err)
		assert.InDelta(t, tc.want, res, 1e-9, tc.dp)
	}

	_, err := plugValue(dps, "99", 1)
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}
