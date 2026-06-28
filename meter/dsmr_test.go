package meter

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/obis"
	"github.com/stretchr/testify/require"
)

// dsmrTelegram40 is the canonical DSMR 4.0 example telegram (nlrb/node-dsmr-parser
// example-telegram.txt). Its published CRC F46A — over the frame up to and
// including '!' with CRLF line endings — is an independent reference vector for
// crc16ARC. It carries multi-value, gas and intentionally malformed lines and no
// per-phase or combined energy registers, so no phase capability is offered.
const dsmrTelegram40 = `/ISk5\2MT382-1000

1-3:0.2.8(40)
0-0:1.0.0(101209113020W)
0-0:96.1.1(4B384547303034303436333935353037)
1-0:1.8.1(123456.789*kWh)
1-0:1.8.2(123456.789*kWh)
1-0:2.8.1(123456.789*kWh)
1-0:2.8.2(123456.789*kWh)
0-0:96.14.0(0002)
1-0:1.7.0(01.193*kW)
1-0:2.7.0(00.000*kW)
0-0:17.0.0(016.1*kW)
0-0:96.3.10(1)
0-0:96.7.21(00004)
0-0:96.7.9(00002)
1-0:99:97.0(2)(0:96.7.19)(101208152415W)(0000000240*s)(101208151004W)(00000000301*s)
1-0:32.32.0(00002)
1-0:52.32.0(00001)
1-0:72:32.0(00000)
1-0:32.36.0(00000)
1-0:52.36.0(00003)
1-0:72.36.0(00000)
0-0:96.13.1(3031203631203831)
0-0:96.13.0(303132333435363738393A3B3C3D3E3F303132333435363738393A3B3C3D3E3F303132333435363738393A3B3C3D3E3F303132333435363738393A3B3C3D3E3F303132333435363738393A3B3C3D3E3F)
0-1:24.1.0(03)
0-1:96.1.0(3232323241424344313233343536373839)
0-1:24.2.1(101209110000W)(12785.123*m3)
0-1:24.4.0(1)
!`

// dsmrTelegram50 is a real DSMR 5.0 telegram (nlrb/node-dsmr-parser
// example-v5.txt) with published CRC 3F8B. It carries all three phases of
// voltage, current and power, so every phase capability is offered, and energy
// via tariff registers (exercising the fallback).
const dsmrTelegram50 = `/ISK5\2M550T-1011

1-3:0.2.8(50)
0-0:1.0.0(170531201444S)
0-0:96.1.1(4530303334303036383234393130313137)
1-0:1.8.1(000000.626*kWh)
1-0:1.8.2(000002.564*kWh)
1-0:2.8.1(000000.000*kWh)
1-0:2.8.2(000000.000*kWh)
0-0:96.14.0(0002)
1-0:1.7.0(00.532*kW)
1-0:2.7.0(00.000*kW)
0-0:96.7.21(00004)
0-0:96.7.9(00002)
1-0:99.97.0()
1-0:32.32.0(00000)
1-0:52.32.0(00000)
1-0:72.32.0(00000)
1-0:32.36.0(00001)
1-0:52.36.0(00001)
1-0:72.36.0(00001)
0-0:96.13.0()
1-0:32.7.0(236.8*V)
1-0:52.7.0(236.1*V)
1-0:72.7.0(238.4*V)
1-0:31.7.0(000*A)
1-0:51.7.0(001*A)
1-0:71.7.0(001*A)
1-0:21.7.0(00.056*kW)
1-0:41.7.0(00.270*kW)
1-0:61.7.0(00.206*kW)
1-0:22.7.0(00.000*kW)
1-0:42.7.0(00.000*kW)
1-0:62.7.0(00.000*kW)
0-1:24.1.0(003)
0-1:96.1.0(4730303332353635353039393034313137)
0-1:24.2.1(170531201008S)(00000.038*m3)
!`

// dsmrTelegramADN is a real telegram from an ADN9 meter with published CRC
// 759E. It is the only example carrying combined 1.8.0/2.8.0 energy registers
// (exercising the non-fallback path) alongside reactive-power registers and all
// three phases.
const dsmrTelegramADN = `/ADN9 6534

0-0:1.0.0(240108164940W)
1-0:1.8.0(00010117.368*kWh)
1-0:2.8.0(00000000.000*kWh)
1-0:3.8.0(00001759.974*kVArh)
1-0:4.8.0(00001260.519*kVArh)
1-0:1.7.0(0004.251*kW)
1-0:2.7.0(0000.000*kW)
1-0:3.7.0(0001.279*kVAr)
1-0:4.7.0(0000.000*kVAr)
1-0:21.7.0(0001.462*kW)
1-0:22.7.0(0000.000*kW)
1-0:41.7.0(0001.263*kW)
1-0:42.7.0(0000.000*kW)
1-0:61.7.0(0001.523*kW)
1-0:62.7.0(0000.000*kW)
1-0:23.7.0(0000.529*kVAr)
1-0:24.7.0(0000.000*kVAr)
1-0:43.7.0(0000.500*kVAr)
1-0:44.7.0(0000.000*kVAr)
1-0:63.7.0(0000.269*kVAr)
1-0:64.7.0(0000.000*kVAr)
1-0:32.7.0(221.2*V)
1-0:52.7.0(224.3*V)
1-0:72.7.0(223.3*V)
1-0:31.7.0(007.0*A)
1-0:51.7.0(006.0*A)
1-0:71.7.0(006.9*A)
!`

// dsmrTelegramExport is a synthetic DSMR 4.2 telegram in which the meter is
// exporting: total and per-phase L3 export power are non-zero while import is
// zero. It exercises the sign-correction path no real example covers — negative
// CurrentPower, a negative L3 phase current (negated because 62.7.0 > 0) and a
// negative L3 phase power. Hand-crafted, so the CRC is computed, not published.
const dsmrTelegramExport = `/XMX5XMXAA10012039345

1-3:0.2.8(42)
0-0:1.0.0(260604130000W)
0-0:96.1.1(4530303539303030353433383330383031)
1-0:1.8.1(004512.345*kWh)
1-0:1.8.2(003241.123*kWh)
1-0:2.8.1(001234.567*kWh)
1-0:2.8.2(000453.210*kWh)
0-0:96.14.0(0002)
1-0:1.7.0(00.000*kW)
1-0:2.7.0(02.450*kW)
0-0:96.7.21(00002)
0-0:96.7.9(00000)
1-0:32.7.0(239.5*V)
1-0:52.7.0(241.1*V)
1-0:72.7.0(238.9*V)
1-0:31.7.0(005.1*A)
1-0:51.7.0(000.0*A)
1-0:71.7.0(004.8*A)
1-0:21.7.0(00.000*kW)
1-0:22.7.0(00.000*kW)
1-0:41.7.0(00.000*kW)
1-0:42.7.0(00.000*kW)
1-0:61.7.0(00.000*kW)
1-0:62.7.0(02.450*kW)
0-1:24.1.0(003)
0-1:24.2.1(260604124500S)(00845.123*m3)
!`

// dsmrFrame returns the telegram with a valid trailing CRC line, as a meter
// would receive it on the wire.
func dsmrFrame(telegram string) []byte {
	frame := []byte(telegram)
	crc := fmt.Sprintf("%04X", crc16ARC(frame))
	return append(frame, []byte(crc+"\r\n")...)
}

// serveTCP streams payload on every accepted connection until the test ends.
func serveTCP(t *testing.T, payload []byte) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				for {
					if _, err := conn.Write(payload); err != nil {
						return
					}
					time.Sleep(50 * time.Millisecond)
				}
			}()
		}
	}()

	return ln.Addr().String()
}

// assertReadings checks the values decoded from dsmrTelegram50.
func assertReadings(t *testing.T, m api.Meter) {
	t.Helper()

	p, err := m.CurrentPower()
	require.NoError(t, err)
	require.Equal(t, 532.0, p)

	me, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok, "MeterEnergy expected (tariff fallback)")
	e, err := me.TotalEnergy()
	require.NoError(t, err)
	require.InDelta(t, 3.190, e, 1e-6) // 1.8.1 + 1.8.2

	re, ok := api.Cap[api.MeterReturnEnergy](m)
	require.True(t, ok, "MeterReturnEnergy expected (tariff fallback)")
	r, err := re.ReturnEnergy()
	require.NoError(t, err)
	require.Equal(t, 0.0, r)

	// all three phases are present, so every phase capability is offered
	pc, ok := api.Cap[api.PhaseCurrents](m)
	require.True(t, ok, "PhaseCurrents expected with all three phases")
	i1, i2, i3, err := pc.Currents()
	require.NoError(t, err)
	require.Equal(t, []float64{0, 1, 1}, []float64{i1, i2, i3})

	pv, ok := api.Cap[api.PhaseVoltages](m)
	require.True(t, ok, "PhaseVoltages expected with all three phases")
	u1, u2, u3, err := pv.Voltages()
	require.NoError(t, err)
	require.Equal(t, []float64{236.8, 236.1, 238.4}, []float64{u1, u2, u3})

	pp, ok := api.Cap[api.PhasePowers](m)
	require.True(t, ok, "PhasePowers expected with all three phases")
	w1, w2, w3, err := pp.Powers()
	require.NoError(t, err)
	require.Equal(t, []float64{56, 270, 206}, []float64{w1, w2, w3})
}

func TestDsmrTCP(t *testing.T) {
	addr := serveTCP(t, dsmrFrame(dsmrTelegram50))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := NewDsmr(ctx, addr, time.Second)
	require.NoError(t, err)

	assertReadings(t, m)
}

func TestDsmrWebSocket(t *testing.T) {
	payload := dsmrFrame(dsmrTelegram50)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		for {
			if err := conn.Write(r.Context(), websocket.MessageText, payload); err != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer srv.Close()

	uri := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := NewDsmr(ctx, uri, time.Second)
	require.NoError(t, err)

	assertReadings(t, m)
}

// TestDsmrIgnoresGarbage verifies that leading garbage and a CRC-mismatched
// frame are skipped while a subsequent valid frame is still parsed.
func TestDsmrIgnoresGarbage(t *testing.T) {
	var payload []byte
	payload = append(payload, []byte("some-garbage-bytes\r\n")...) // not a frame start
	payload = append(payload, []byte(dsmrTelegram40)...)           // frame ...
	payload = append(payload, []byte("0000\r\n")...)               // ... with wrong CRC
	payload = append(payload, dsmrFrame(dsmrTelegram40)...)        // valid frame

	addr := serveTCP(t, payload)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := NewDsmr(ctx, addr, time.Second)
	require.NoError(t, err)

	p, err := m.CurrentPower()
	require.NoError(t, err)
	require.Equal(t, 1193.0, p)
}

// TestDsmrConstructTimeoutStopsReader verifies that when the initial frame does
// not arrive before the constructor timeout, canceling the context (as the
// device setup does on a constructor error) stops the background read loop and
// closes the connection instead of leaking it.
func TestDsmrConstructTimeoutStopsReader(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	closed := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// never send a frame; the read returns once the client (run) closes on cancel
		_, _ = conn.Read(make([]byte, 1))
		close(closed)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err = NewDsmr(ctx, ln.Addr().String(), 200*time.Millisecond)
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)

	// device setup cancels the context on a constructor error
	cancel()

	select {
	case <-closed:
		// run closed the connection => the goroutine is terminating
	case <-time.After(2 * time.Second):
		t.Fatal("reader connection was not closed after context cancel")
	}
}

// TestDsmrCRC validates crc16ARC against the published reference CRCs of two
// real example telegrams. The wire format uses CRLF line endings and the
// checksum covers the frame up to and including '!'.
func TestDsmrCRC(t *testing.T) {
	for _, tc := range []struct {
		name     string
		telegram string
		crc      uint16
	}{
		{"v4.0", dsmrTelegram40, 0xF46A},
		{"v5.0", dsmrTelegram50, 0x3F8B},
		{"adn9", dsmrTelegramADN, 0x759E},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame := strings.ReplaceAll(tc.telegram, "\n", "\r\n")
			require.Equal(t, tc.crc, crc16ARC([]byte(frame)))
		})
	}
}

// newTestDsmr builds a Dsmr around a fixed frame, bypassing the transport, to
// unit-test the capability decoration and register-selection logic.
func newTestDsmr(frame map[string]string) *Dsmr {
	m := &Dsmr{
		Caps:    implement.New(),
		timeout: time.Minute,
		updated: time.Now(),
		frame:   frame,
	}
	m.decorateEnergy()
	m.decorateCurrents()
	m.decorateVoltages()
	m.decoratePowers()
	return m
}

// TestDsmrADN reads the real ADN9 telegram end-to-end: combined 1.8.0/2.8.0
// energy registers and all three phases, decoded from a telegram interleaved
// with reactive-power registers the parser must ignore.
func TestDsmrADN(t *testing.T) {
	addr := serveTCP(t, dsmrFrame(dsmrTelegramADN))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := NewDsmr(ctx, addr, time.Second)
	require.NoError(t, err)

	p, err := m.CurrentPower()
	require.NoError(t, err)
	require.Equal(t, 4251.0, p)

	me, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok, "MeterEnergy expected (combined register)")
	e, err := me.TotalEnergy()
	require.NoError(t, err)
	require.InDelta(t, 10117.368, e, 1e-6)

	re, ok := api.Cap[api.MeterReturnEnergy](m)
	require.True(t, ok, "MeterReturnEnergy expected (combined register)")
	r, err := re.ReturnEnergy()
	require.NoError(t, err)
	require.Equal(t, 0.0, r)

	pc, ok := api.Cap[api.PhaseCurrents](m)
	require.True(t, ok, "PhaseCurrents expected with all three phases")
	i1, i2, i3, err := pc.Currents()
	require.NoError(t, err)
	require.Equal(t, []float64{7.0, 6.0, 6.9}, []float64{i1, i2, i3})

	pv, ok := api.Cap[api.PhaseVoltages](m)
	require.True(t, ok, "PhaseVoltages expected with all three phases")
	u1, u2, u3, err := pv.Voltages()
	require.NoError(t, err)
	require.Equal(t, []float64{221.2, 224.3, 223.3}, []float64{u1, u2, u3})

	pp, ok := api.Cap[api.PhasePowers](m)
	require.True(t, ok, "PhasePowers expected with all three phases")
	w1, w2, w3, err := pp.Powers()
	require.NoError(t, err)
	require.Equal(t, []float64{1462, 1263, 1523}, []float64{w1, w2, w3})
}

// TestDsmrExport reads an exporting meter and checks the sign-correction path:
// negative total power, and a negative L3 phase current and power (L3 export
// register 62.7.0 is non-zero), while the non-exporting phases stay positive.
func TestDsmrExport(t *testing.T) {
	addr := serveTCP(t, dsmrFrame(dsmrTelegramExport))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := NewDsmr(ctx, addr, time.Second)
	require.NoError(t, err)

	p, err := m.CurrentPower()
	require.NoError(t, err)
	require.Equal(t, -2450.0, p)

	// energy via tariff fallback
	me, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok, "MeterEnergy expected (tariff fallback)")
	e, err := me.TotalEnergy()
	require.NoError(t, err)
	require.InDelta(t, 7753.468, e, 1e-6) // 1.8.1 + 1.8.2

	re, ok := api.Cap[api.MeterReturnEnergy](m)
	require.True(t, ok, "MeterReturnEnergy expected (tariff fallback)")
	r, err := re.ReturnEnergy()
	require.NoError(t, err)
	require.InDelta(t, 1687.777, r, 1e-6) // 2.8.1 + 2.8.2

	// L3 exports, so its current and power are negative; L1/L2 stay positive
	pc, ok := api.Cap[api.PhaseCurrents](m)
	require.True(t, ok, "PhaseCurrents expected with all three phases")
	i1, i2, i3, err := pc.Currents()
	require.NoError(t, err)
	require.Equal(t, []float64{5.1, 0.0, -4.8}, []float64{i1, i2, i3})

	pp, ok := api.Cap[api.PhasePowers](m)
	require.True(t, ok, "PhasePowers expected with all three phases")
	w1, w2, w3, err := pp.Powers()
	require.NoError(t, err)
	require.Equal(t, []float64{0, 0, -2450}, []float64{w1, w2, w3})
}

// TestDsmrCombinedEnergy verifies the combined 1.8.0/2.8.0 registers are
// preferred over the per-tariff sum. No real example carries both, so the
// preference is covered directly.
func TestDsmrCombinedEnergy(t *testing.T) {
	m := newTestDsmr(map[string]string{
		obis.EnergyImport:   "1234.567", // combined; tariffs below sum to a different value
		obis.EnergyImportT1: "100.000",
		obis.EnergyImportT2: "200.000",
		obis.EnergyExport:   "89.012",
	})

	me, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok, "MeterEnergy expected")
	e, err := me.TotalEnergy()
	require.NoError(t, err)
	require.InDelta(t, 1234.567, e, 1e-6)

	re, ok := api.Cap[api.MeterReturnEnergy](m)
	require.True(t, ok, "MeterReturnEnergy expected")
	r, err := re.ReturnEnergy()
	require.NoError(t, err)
	require.InDelta(t, 89.012, r, 1e-6)
}

// TestDsmrPhaseGating verifies the three-phase capabilities are only offered
// when all three phases are present, not for a partial (single-phase) frame.
func TestDsmrPhaseGating(t *testing.T) {
	m := newTestDsmr(map[string]string{
		obis.CurrentL1: "2", // only L1 present for currents/voltages
		obis.VoltageL1: "230.0",
	})

	_, ok := api.Cap[api.PhaseCurrents](m)
	require.False(t, ok, "PhaseCurrents must not be offered with only L1")
	_, ok = api.Cap[api.PhaseVoltages](m)
	require.False(t, ok, "PhaseVoltages must not be offered with only L1")
	_, ok = api.Cap[api.PhasePowers](m)
	require.False(t, ok, "PhasePowers must not be offered without per-phase power")
}
