package homewizard

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test StateResponse
func TestUnmarshalStateResponse(t *testing.T) {
	{
		// Shelly 1 PM channel 0 (1)
		var res StateResponse

		jsonstr := `{"power_on": true,"switch_lock": false,"brightness": 255}`
		assert.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, true, res.PowerOn)
	}
}
