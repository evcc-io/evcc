package eebus

import (
	"testing"
	"time"

	"github.com/enbility/ship-go/cert"
	"github.com/evcc-io/evcc/hems/eebus"
	hems "github.com/evcc-io/evcc/hems/eebus"
	server "github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

// TestControlBoxGridGuardHeartbeat covers issue #31561: a real ControlBox exposes LPC/LPP
// only on a GridGuard entity; if eebus-go's cs/lpc validEntityTypes ever drops it again, the heartbeat subscription silently never happens and the LPC failsafe eventually trips.
func TestControlBoxGridGuardHeartbeat(t *testing.T) {
	util.LogLevel("error", map[string]string{"eebus": "trace"})

	// tear down a server instance left over from other tests in this package
	if inst, err := server.Instance(); err == nil {
		inst.Shutdown()
	}

	certificate, err := cert.CreateCertificate("Demo", "Demo", "DE", "Demo-GridGuard-01")
	require.NoError(t, err, "certificate")

	public, private, err := server.GetX509KeyPair(certificate)
	require.NoError(t, err, "decode certificate")

	_, err = server.NewServer(server.Config{
		Port: freePort(t),
		Certificate: server.Certificate{
			Public:  public,
			Private: private,
		},
	})
	require.NoError(t, err, "server")

	inst, err := server.Instance()
	require.NoError(t, err, "instance")
	defer inst.Shutdown()

	// the controlbox test double, like a real ControlBox, only exposes GridGuard
	box, err := createControlbox(t.Context(), inst.Ski(), freePort(t))
	require.NoError(t, err, "controlbox")
	defer box.myService.Shutdown()

	// no hems.Run(): the run loop applies limits via the nil site and is not needed here
	_, err = hems.NewEEBus(t.Context(), box.ski, eebus.Limits{}, nil, nil, time.Second)
	require.NoError(t, err, "hems")

	require.Eventually(t, inst.ControllableSystem().CsLPCInterface.IsHeartbeatWithinDuration,
		10*time.Second, 100*time.Millisecond,
		"CsLPC never subscribed to the ControlBox's GridGuard heartbeat")
}
