package octopusde

import (
	"testing"

	"github.com/hasura/go-graphql-client/pkg/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecodeDevices verifies the GraphQL response decodes through the same
// unmarshaller the client uses, covering the SmartFlex status inline fragments.
func TestDecodeDevices(t *testing.T) {
	data := []byte(`{
		"devices": [
			{
				"id": "dev-1",
				"name": "My Car",
				"deviceType": "ELECTRIC_VEHICLES",
				"provider": "TESLA",
				"status": {
					"stateOfCharge": {"value": 55},
					"stateOfChargeLimit": {"upperSocLimit": 80}
				}
			},
			{
				"id": "dev-2",
				"name": "Wallbox",
				"deviceType": "CHARGE_POINTS",
				"provider": "OCPP",
				"status": {
					"stateOfCharge": {"value": 42}
				}
			},
			{
				"id": "dev-3",
				"name": "No Soc",
				"deviceType": "CHARGE_POINTS",
				"provider": "OCPP",
				"status": {}
			}
		]
	}`)

	var q krakenDevices
	require.NoError(t, jsonutil.UnmarshalGraphQL(data, &q))
	require.Len(t, q.Devices, 3)

	// vehicle with soc and target limit
	veh := q.Devices[0]
	assert.Equal(t, "dev-1", veh.ID)
	assert.Equal(t, "My Car", veh.Name)
	soc, ok := veh.Soc()
	assert.True(t, ok)
	assert.Equal(t, float64(55), soc)
	limit, ok := veh.TargetSoc()
	assert.True(t, ok)
	assert.Equal(t, float64(80), limit)

	// charge point with soc but no target limit
	cp := q.Devices[1]
	soc, ok = cp.Soc()
	assert.True(t, ok)
	assert.Equal(t, float64(42), soc)
	_, ok = cp.TargetSoc()
	assert.False(t, ok)

	// device without any state of charge
	none := q.Devices[2]
	_, ok = none.Soc()
	assert.False(t, ok)
}
