package eebus

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/enbility/eebus-go/usecases/eg/lpc"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/cert"
	hems "github.com/evcc-io/evcc/hems/eebus"
	"github.com/evcc-io/evcc/server/db/settings"
	server "github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

// freePort returns a currently unused tcp port
func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

// qrField extracts a field value from the SHIP QR code text
func qrField(qr, key string) string {
	for f := range strings.SplitSeq(qr, ";") {
		if v, ok := strings.CutPrefix(f, key+":"); ok {
			return v
		}
	}
	return ""
}

// serverQR returns the SHIP QR code text of the running eebus server
func serverQR(t *testing.T) string {
	t.Helper()

	b, err := json.Marshal(server.GetStatus())
	require.NoError(t, err)

	var status struct{ QR string }
	require.NoError(t, json.Unmarshal(b, &status))

	return status.QR
}

// createPairingControlbox creates a controlbox that pairs with the remote service
// via the SHIP Pairing Service (announcer mode) using the scanned QR code text
func createPairingControlbox(ctx context.Context, qr string, port int) (*controlbox, error) {
	secret, err := hex.DecodeString(qrField(qr, "SPSEC"))
	if err != nil {
		return nil, err
	}

	target := shipapi.PairingTarget{
		SKI:         qrField(qr, "SKI"),
		Fingerprint: qrField(qr, "FPH256"),
		ShipID:      qrField(qr, "ID"),
		Secret:      secret,
	}

	h, err := newControlbox(shipapi.NewPairingConfig(shipapi.PairingModeAnnouncer, nil), port)
	if err != nil {
		return nil, err
	}

	// devZ must trust devA before announcing
	h.myService.RegisterRemoteService(shipapi.NewServiceIdentity(target.SKI, target.Fingerprint, target.ShipID))
	h.start(ctx)

	// hub start is asynchronous- retry until announcing
	for range 10 {
		if err = h.myService.StartAnnouncementTo(target); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	return h, err
}

// connectHems creates the hems consumer for ski (empty ski = SHIP-paired device)
// and returns the connect result asynchronously
func connectHems(ctx context.Context, ski string) <-chan error {
	errC := make(chan error, 1)
	go func() {
		_, err := hems.NewEEBus(ctx, ski, hems.Limits{}, nil, nil, time.Second)
		errC <- err
	}()
	return errC
}

// findPairing returns the first pairing matching source
func findPairing(pairings []server.PairingInfo, source server.PairingSource) (server.PairingInfo, bool) {
	for _, p := range pairings {
		if p.Source == source {
			return p, true
		}
	}
	return server.PairingInfo{}, false
}

func TestShipPairing(t *testing.T) {
	util.LogLevel("error", map[string]string{"eebus": "trace"})

	// tear down a server instance left over from other tests in this package
	if inst, err := server.Instance(); err == nil {
		inst.Shutdown()
	}

	certificate, err := cert.CreateCertificate("Demo", "Demo", "DE", "Demo-Pairing-01")
	require.NoError(t, err, "certificate")

	public, private, err := server.GetX509KeyPair(certificate)
	require.NoError(t, err, "decode certificate")

	secret, err := server.CreatePairingSecret()
	require.NoError(t, err, "secret")

	conf := server.Config{
		Port:   freePort(t),
		Secret: secret,
		Certificate: server.Certificate{
			Public:  public,
			Private: private,
		},
	}

	_, err = server.NewServer(conf)
	require.NoError(t, err, "server")

	inst, err := server.Instance()
	require.NoError(t, err, "instance")

	qr := serverQR(t)
	require.Contains(t, qr, "SPSEC:", "expected SHIP Pairing Service QR")

	// consumer for the SHIP-paired controlbox must connect once pairing completes
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	hemsC := connectHems(ctx, "")

	box, err := createPairingControlbox(t.Context(), qr, freePort(t))
	require.NoError(t, err, "controlbox")

	require.NoError(t, <-hemsC, "paired device not routed to consumer")

	// LPC data flows to the controlbox
	require.Eventually(t, func() bool {
		return len(box.remoteEntity(lpc.DataUpdateLimit)) > 0
	}, 30*time.Second, 100*time.Millisecond, "waiting for lpc.DataUpdateLimit")

	// trusted device identity is persisted
	var identities []shipapi.ServiceIdentity
	require.NoError(t, settings.Json("eebus.pairing.trusted", &identities), "trusted device not persisted")
	require.Len(t, identities, 1, "trusted device not persisted")
	require.NotEmpty(t, identities[0].SKI, "paired device ski not persisted")

	// restart evcc- the paired device must reconnect without re-pairing
	inst.Shutdown()

	_, err = server.NewServer(conf)
	require.NoError(t, err, "server restart")

	inst, err = server.Instance()
	require.NoError(t, err, "instance restart")

	ctx2, cancel2 := context.WithTimeout(t.Context(), time.Minute)
	defer cancel2()
	require.NoError(t, <-connectHems(ctx2, ""), "paired device not reconnected after restart")

	// a device trusted by configured SKI must be listed too, tagged accordingly,
	// and must not be removable via RemovePairing
	box2, err := createControlbox(t.Context(), inst.Ski(), freePort(t))
	require.NoError(t, err, "ski-configured controlbox")

	ctx3, cancel3 := context.WithTimeout(t.Context(), time.Minute)
	defer cancel3()
	require.NoError(t, <-connectHems(ctx3, box2.ski), "ski-configured device not connected")

	pairings := inst.Pairings()
	require.Len(t, pairings, 2)

	pairedEntry, ok := findPairing(pairings, server.PairingSourcePaired)
	require.True(t, ok, "paired entry missing")
	skiEntry, ok := findPairing(pairings, server.PairingSourceSki)
	require.True(t, ok, "ski entry missing")
	require.Equal(t, box2.ski, skiEntry.SKI)

	require.False(t, inst.RemovePairing(skiEntry.SKI), "ski-configured pairing must not be removable")
	require.Len(t, inst.Pairings(), 2)

	// remove the individual pairing- trust is revoked and persisted state cleared
	require.True(t, inst.RemovePairing(pairedEntry.ShipID), "pairing not removed")
	require.Len(t, inst.Pairings(), 1)

	require.NoError(t, settings.Json("eebus.pairing.trusted", &identities))
	require.Empty(t, identities, "removed pairing still persisted")
}
