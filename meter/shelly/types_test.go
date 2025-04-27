package shelly

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Shelly device info
func TestUnmarshalDeviceInfoResponse(t *testing.T) {
	{
		// Shelly Pro 3EM
		var res DeviceInfo

		jsonstr := `{"name":null,"id":"shellypro3em-fce8c0dba900","mac":"FCE8C0DBA900","slot":1,"model":"SPEM-003CEBEU","gen":2,"fw_id":"20241011-114455/1.4.4-g6d2a586","ver":"1.4.4","app":"Pro3EM","auth_en":false,"auth_domain":null,"profile":"monophase"}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, 2, res.Gen)
		assert.Equal(t, "SPEM-003CEBEU", res.Model)
		assert.Equal(t, "monophase", res.Profile)
	}
}
