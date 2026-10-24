package ecoflow

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deviceServer answers the app device list with body.
func deviceServer(t *testing.T, body string) *Session {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/iot-service/user/device" {
			http.NotFound(w, r)
			return
		}
		assert.Equal(t, "Bearer app-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return &Session{UserID: "42", Base: srv.URL, Token: "app-token"}
}

func TestGatewayFindsTheOnlyPowerOcean(t *testing.T) {
	session := deviceServer(t, `{"code":"0","data":{
		"bound":[
			{"sn":"C376000000000001","deviceName":"PowerPulse 2","productType":91},
			{"sn":"HJ35000000000001","deviceName":"PowerOcean","productType":83},
			{"sn":"R351000000000001","deviceName":"River 2 Pro","productType":13}
		],
		"share":[]
	}}`)

	gateway, err := Gateway(util.NewLogger("test"), session)
	require.NoError(t, err)
	assert.Equal(t, "HJ35000000000001", gateway)
}

func TestGatewayFindsPowerOceanInSerialKeyedGroups(t *testing.T) {
	session := deviceServer(t, `{"code":"0","data":{
		"bound":{"C376000000000001":{"deviceName":"PowerPulse 2"}},
		"share":{"hj31000000000002":{"deviceName":"PowerOcean"}}
	}}`)

	gateway, err := Gateway(util.NewLogger("test"), session)
	require.NoError(t, err)
	assert.Equal(t, "HJ31000000000002", gateway)
}

func TestGatewayFailsWithoutPowerOcean(t *testing.T) {
	session := deviceServer(t, `{"code":"0","data":{"bound":[{"sn":"C376000000000001"}],"share":[]}}`)

	_, err := Gateway(util.NewLogger("test"), session)
	require.Error(t, err)
	assert.ErrorContains(t, err, "no PowerOcean")
	assert.ErrorContains(t, err, "gateway")
	assert.ErrorContains(t, err, "C376000000000001", "names the devices it saw")
}

func TestGatewayFailsWithSeveralPowerOceans(t *testing.T) {
	session := deviceServer(t, `{"code":"0","data":{
		"bound":[{"sn":"HJ35000000000001"},{"sn":"C376000000000001"}],
		"share":[{"sn":"HJ31000000000002"},{"sn":"HJ35000000000001"}]
	}}`)

	_, err := Gateway(util.NewLogger("test"), session)
	require.Error(t, err)
	assert.ErrorContains(t, err, "HJ31000000000002, HJ35000000000001")
	assert.ErrorContains(t, err, "gateway")
}

func TestGatewayReportsApiError(t *testing.T) {
	session := deviceServer(t, `{"code":"401","message":"token expired"}`)

	_, err := Gateway(util.NewLogger("test"), session)
	assert.ErrorContains(t, err, "token expired")
}

func TestConnectionWithoutLoginHasNoGateway(t *testing.T) {
	conn := NewConnection(util.NewLogger("test"), nil, "42")

	_, err := conn.Gateway()
	assert.Error(t, err)
}
