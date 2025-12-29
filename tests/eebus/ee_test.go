package eebus

import (
	"testing"
	"time"

	"github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/ship-go/cert"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/hems/eebus"
	hems "github.com/evcc-io/evcc/hems/eebus"
	server "github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

const remotePort = 9001

func TestEEBus(t *testing.T) {
	certificate, err := cert.CreateCertificate("Demo", "Demo", "DE", "Demo-Server-01")
	require.NoError(t, err, "certificate")

	public, private, err := server.GetX509KeyPair(certificate)
	require.NoError(t, err, "decode certificate")

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

	hems, err := hems.NewEEBus(t.Context(), box.ski, eebus.Limits{}, lpcCircuit, lppCircuit, time.Second)
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

	// TODO no error
	require.Error(t, err, "consumption limit")
}
