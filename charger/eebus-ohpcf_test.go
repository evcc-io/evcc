package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Status must surface a not-connected error rather than reporting StatusA,
// so a missing compressor is not mistaken for an idle device.
func TestEEBusOHPCFStatusNotConnected(t *testing.T) {
	c := &EEBusOHPCF{}

	status, err := c.Status()
	require.Error(t, err)
	assert.Equal(t, api.StatusNone, status)
}
