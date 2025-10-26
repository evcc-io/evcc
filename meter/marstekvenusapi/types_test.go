package marstekvenusapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalErrorResp(t *testing.T) {

	var resp Response

	jsonstr := `{
	"id":	0,
	"src":	"VenusE 3.0-123456789012",
	"error":	{
		"code":	-32700,
		"message":	"Parse error",
		"data":	403
	}
}`
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &resp))
	assert.Nil(t, resp.Result)

	assert.Equal(t, 403, resp.Error.Data)
	assert.Equal(t, -32700, resp.Error.Code)
}

// Test Marstek device info
func TestUnmarshalDeviceInfoResp(t *testing.T) {

	var resp Response
	var reslt DeviceResult

	jsonstr := `{
	"id":	0,
	"src":	"VenusE 3.0-123456789012",
	"result":	{
		"device":	"VenusE 3.0",
		"ver":	139,
		"ble_mac":	"123456789012",
		"wifi_mac":	"098765432109",
		"wifi_name":	"wifi.de",
		"ip":	"192.168.177.183"
	}
}`

	require.NoError(t, json.Unmarshal([]byte(jsonstr), &resp))
	assert.Equal(t, 0, resp.ID)

	require.NoError(t, json.Unmarshal([]byte(resp.Result), &reslt))

	assert.Equal(t, 139, reslt.Version)
}

func TestUnmarshalBatStatusResp(t *testing.T) {
	var resp Response
	var reslt BatStatusResult

	jsonstr := `{
	"id":	4,
	"src":	"VenusE 3.0-123456789012",
	"result":	{
		"id":	0,
		"soc":	12,
		"charg_flag":	true,
		"dischrg_flag":	true,
		"bat_temp":	211.0,
		"bat_capacity":	61.0,
		"rated_capacity":	5120.0
	}
}`
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &resp))
	require.NoError(t, json.Unmarshal([]byte(resp.Result), &reslt))
	assert.Equal(t, 12, reslt.StateOfCharge)
}
