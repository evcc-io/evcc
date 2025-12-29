package eebus

import (
	"testing"
	"time"

	"github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/hems/eebus"
	server "github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

var (
	public = `-----BEGIN CERTIFICATE-----
MIIBvDCCAWOgAwIBAgIRANIxZf/UYNXTaeo2eSd0d/8wCgYIKoZIzj0EAwIwPjEL
MAkGA1UEBhMCREUxDTALBgNVBAoTBEVWQ0MxCTAHBgNVBAsTADEVMBMGA1UEAwwM
RVZDQ19IRU1TXzAxMB4XDTI1MTIyODEyNDAwNFoXDTM1MTIyNjEyNDAwNFowPjEL
MAkGA1UEBhMCREUxDTALBgNVBAoTBEVWQ0MxCTAHBgNVBAsTADEVMBMGA1UEAwwM
RVZDQ19IRU1TXzAxMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFU5Mc01A1hTv
dY6GyaRrTiOOnHaNa1ZNlzfU9ZofoxSWNkTetbuG+/KdcJ881H2t3EnBHurQfLnw
bBfHzpLhKaNCMEAwDgYDVR0PAQH/BAQDAgeAMA8GA1UdEwEB/wQFMAMBAf8wHQYD
VR0OBBYEFJBx687g+7HkDQB8mFsEImUWOLx/MAoGCCqGSM49BAMCA0cAMEQCIAgS
D6kBneohzMbkF1M7VITk/Sbk/imgdI8wvr1O8bzbAiBl9MkNXUM8zQOF5kQf26H2
hNTc0g23cSuUxO61ZKAKyg==
-----END CERTIFICATE-----`

	private = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMkBbXxvAnFrRK93h2OzpOeVG5CIJEr0ErO8VlJufu8EoAoGCCqGSM49
AwEHoUQDQgAEFU5Mc01A1hTvdY6GyaRrTiOOnHaNa1ZNlzfU9ZofoxSWNkTetbuG
+/KdcJ881H2t3EnBHurQfLnwbBfHzpLhKQ==
-----END EC PRIVATE KEY-----`
)

const remotePort = 9001

func TestEEBus(t *testing.T) {
	srv, err := server.NewServer(server.Config{
		Certificate: server.Certificate{
			Public:  public,
			Private: private,
		},
	})

	require.NoError(t, err, "server")
	require.NotEmpty(t, srv.Ski, "server ski")

	server.Instance = srv
	go srv.Run()

	lpcCircuit, err := circuit.New(util.NewLogger("lpc"), "lpc", 0, 0, nil, time.Minute)
	require.NoError(t, err)

	lppCircuit, err := circuit.New(util.NewLogger("lpp"), "lpp", 0, 0, nil, time.Minute)
	require.NoError(t, err)

	box, err := createControlbox(t.Context(), server.Instance.Ski, remotePort)
	require.NoError(t, err, "controlbox")

	eventC := make(chan api.EventType, 1)
	box.remoteEventC = eventC

	hems, err := eebus.NewEEBus(t.Context(), box.ski, eebus.Limits{}, lpcCircuit, lppCircuit, time.Second)
	require.NoError(t, err, "hems")

	go hems.Run()

	<-eventC
	t.Log(box.remoteEntities)
	srvEntity := box.remoteEntity(lpc.UseCaseSupportUpdate)[0]

	_, err = box.uclpc.WriteConsumptionLimit(srvEntity, ucapi.LoadLimit{
		IsActive: true,
		Value:    1,
	}, func(result model.ResultDataType) {
		t.Logf("lpc result: %v", result)
	})
	require.NoError(t, err, "consumption limit")

	t.Error("foo")
}
