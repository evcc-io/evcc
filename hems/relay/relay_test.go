package relay

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConnectedFailsafe verifies that Connected/Failsafe make no statement for
// the Relay HEMS — these are EEBus-specific and must be nil so the API/MQTT
// output omits them for relay-based setups.
func TestConnectedFailsafe(t *testing.T) {
	c := &Relay{}
	require.Nil(t, c.Connected())
	require.Nil(t, c.Failsafe())
}
