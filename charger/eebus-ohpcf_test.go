package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// methods accessing the compressor entity must error when it is absent,
// so a missing compressor is not mistaken for an idle device.
func TestEEBusOHPCFNotConnected(t *testing.T) {
	c := &EEBusOHPCF{}

	status, err := c.Status()
	require.ErrorIs(t, err, errNotConnected)
	assert.Equal(t, api.StatusNone, status)

	_, err = c.Enabled()
	require.ErrorIs(t, err, errNotConnected)

	require.ErrorIs(t, c.Enable(true), errNotConnected)
	require.ErrorIs(t, c.MaxCurrent(16), errNotConnected)
}
