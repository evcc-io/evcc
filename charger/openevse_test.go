package charger

import (
	"testing"

	"github.com/RAR/go-openevse"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenEVSEStatus(t *testing.T) {
	tc := []struct {
		state, vehicle int
		want           api.ChargeStatus // StatusNone means an error is expected
		wantErr        string           // substring expected in the error, when want == StatusNone
	}{
		{1, 0, api.StatusA, ""},                     // not connected
		{2, 1, api.StatusB, ""},                     // connected
		{2, 0, api.StatusA, ""},                     // connected but no vehicle
		{3, 1, api.StatusC, ""},                     // charging
		{4, 1, api.StatusB, ""},                     // vent required, vehicle connected
		{254, 0, api.StatusA, ""},                   // sleeping, unplugged
		{255, 1, api.StatusB, ""},                   // disabled, plugged
		{0, 1, api.StatusNone, "invalid status: 0"}, // unknown
		{6, 1, api.StatusNone, "gfci fault"},
		{8, 1, api.StatusNone, "stuck relay"},
		{11, 1, api.StatusNone, "over current"},
	}

	for _, tt := range tc {
		status, err := openevseStatus(openevse.Status{State: tt.state, Vehicle: tt.vehicle})
		if tt.want == api.StatusNone {
			assert.ErrorContains(t, err, tt.wantErr, "state %d", tt.state)
		} else {
			require.NoError(t, err, "state %d", tt.state)
			assert.Equal(t, tt.want, status, "state %d", tt.state)
		}
	}
}

func TestOpenEVSETag(t *testing.T) {
	assert.Equal(t, "", openevseTag(""))
	assert.Equal(t, "", openevseTag("\x00"))
	assert.Equal(t, "", openevseTag(" \n"))
	assert.Equal(t, "04A1B2C3", openevseTag("04A1B2C3"))
	assert.Equal(t, "04A1B2C3", openevseTag(" 04A1B2C3\x00"))
}
