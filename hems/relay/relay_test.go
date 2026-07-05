package relay

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRelayNoLimitContract verifies api.HEMS's "nil = no limit" contract holds
// for a freshly-constructed Relay before any limit has been set.
func TestRelayNoLimitContract(t *testing.T) {
	c := new(Relay)

	require.Nil(t, c.MaxConsumptionPower())
	require.Nil(t, c.MaxProductionPower())
}
