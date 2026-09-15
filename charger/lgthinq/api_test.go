package lgthinq

import (
	"net/http"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	api := NewAPI(util.NewLogger("test"), "pat", "de")

	// mock below the header decorator
	api.Client.Transport.(*transport.Decorator).Base = httpmock.DefaultTransport
	defer httpmock.Reset()

	assert.Equal(t, "https://api-eic.lgthinq.com", api.base)

	httpmock.RegisterResponder(http.MethodGet, api.base+"/devices/1/state", func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "Bearer pat", req.Header.Get("Authorization"))
		assert.Equal(t, "DE", req.Header.Get("x-country"))
		assert.NotEmpty(t, req.Header.Get("x-message-id"))

		return httpmock.NewStringResponse(http.StatusOK, `{"messageId":"x","timestamp":"y","response":{
			"waterHeaterJobMode":{"currentJobMode":"HEAT_PUMP"},
			"operation":{"waterHeaterOperationMode":"POWER_ON"},
			"temperatureInUnits":[{"currentTemperature":45,"targetTemperature":50,"unit":"C"},{"currentTemperature":113,"targetTemperature":122,"unit":"F"}]
		}}`), nil
	})

	httpmock.RegisterResponder(http.MethodPost, api.base+"/devices/1/control",
		httpmock.NewStringResponder(http.StatusBadRequest, `{"messageId":"x","timestamp":"y","error":{"code":"1207","message":"NOT_SUPPORTED_PRODUCT"}}`))

	var state WaterHeaterState
	require.NoError(t, api.State("1", &state))

	temp, ok := state.Celsius()
	require.True(t, ok)
	assert.Equal(t, 45.0, temp.CurrentTemperature)
	assert.Equal(t, 50.0, temp.TargetTemperature)

	assert.EqualError(t, api.WaterHeaterTargetTemperature("1", 60), "1207: NOT_SUPPORTED_PRODUCT")
}
