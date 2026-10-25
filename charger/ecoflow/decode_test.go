// Ported from shuette42/ecoflow-energy-ha (MIT), see the package documentation.

package ecoflow

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

func varint(b []byte, num protowire.Number, v uint64) []byte {
	b = protowire.AppendTag(b, num, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

func nest(b []byte, num protowire.Number, inner []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, inner)
}

func fixed32(b []byte, num protowire.Number, v float32) []byte {
	b = protowire.AppendTag(b, num, protowire.Fixed32Type)
	return protowire.AppendFixed32(b, math.Float32bits(v))
}

func TestDecodeHeartbeatPacked(t *testing.T) {
	var volts, amps []byte
	for _, v := range []float32{231.6, 231.5, 231.7} {
		volts = protowire.AppendFixed32(volts, math.Float32bits(v))
	}
	for _, a := range []float32{5.95, 5.94, 5.96} {
		amps = protowire.AppendFixed32(amps, math.Float32bits(a))
	}

	var p []byte
	p = varint(p, 1, 3)
	p = varint(p, 18, 160)
	p = fixed32(p, 28, 4070.7)
	p = nest(p, 29, volts)
	p = nest(p, 30, amps)
	p = varint(p, 44, 1372690)

	h, err := DecodeHeartbeat(p)
	require.NoError(t, err)

	assert.Equal(t, LayoutTopLevel, h.Layout)
	assert.InDelta(t, 4070.7, h.PowerW, 0.1)
	require.Len(t, h.Volts, 3)
	require.Len(t, h.Amps, 3)
	assert.InDelta(t, 5.95, h.Amps[0], 0.01)
	assert.Equal(t, uint64(1372690), h.TotalEnergyWh)
}

// The same values may arrive as repeated fixed32 fields instead of one packed
// block; both encodings must yield all three phases.
func TestDecodeHeartbeatRepeatedFixed32(t *testing.T) {
	var p []byte
	p = varint(p, 1, 2)
	for _, v := range []float32{229.3, 0, 0} {
		p = fixed32(p, 29, v)
	}

	h, err := DecodeHeartbeat(p)
	require.NoError(t, err)
	require.Len(t, h.Volts, 3)
	assert.InDelta(t, 229.3, h.Volts[0], 0.01)
}

func TestDecodeHeartbeatPlausibilityGate(t *testing.T) {
	_, err := DecodeHeartbeat(varint(nil, 9, 100))
	assert.Error(t, err, "heartbeat without a state field must be rejected")

	_, err = DecodeHeartbeat(varint(nil, 1, 9999))
	assert.Error(t, err, "heartbeat with an implausible state must be rejected")
}

// A non-finite reading must not reach the meter; the heartbeat is dropped.
func TestDecodeHeartbeatRejectsNonFiniteReadings(t *testing.T) {
	for _, v := range []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		top := fixed32(varint(nil, 1, 3), 28, v)
		_, err := DecodeHeartbeat(top)
		assert.Error(t, err, "top-level power %v", v)

		readings := fixed32(fixed32(nil, 4, 1000), 11, v)
		_, err = DecodeHeartbeat(nest(varint(nil, 1, 3), 8, readings))
		assert.Error(t, err, "nested current %v", v)

		var volts []byte
		for _, u := range []float32{230, v, 230} {
			volts = protowire.AppendFixed32(volts, math.Float32bits(u))
		}
		_, err = DecodeHeartbeat(nest(varint(nil, 1, 3), 29, volts))
		assert.Error(t, err, "packed voltage %v", v)
	}
}

func TestDecodeHeartbeatNestedChargeReadings(t *testing.T) {
	var readings []byte
	readings = fixed32(readings, 4, 4070.7)
	for i, v := range []float32{231.6, 231.5, 231.7} {
		readings = fixed32(readings, protowire.Number(7+i), v)
	}
	for i, a := range []float32{5.95, 5.94, 5.96} {
		readings = fixed32(readings, protowire.Number(10+i), a)
	}

	var p []byte
	p = varint(p, 1, 3)
	p = nest(p, 8, readings)

	h, err := DecodeHeartbeat(p)
	require.NoError(t, err)

	assert.Equal(t, LayoutNested, h.Layout)
	assert.InDelta(t, 4070.7, h.PowerW, 0.1)
	require.Len(t, h.Amps, 3)
	assert.InDelta(t, 5.96, h.Amps[2], 0.01)
}

// When both layouts arrive at once, the nested record wins, whatever order
// the fields come in.
func TestDecodeHeartbeatNestedWinsOverTopLevel(t *testing.T) {
	readings := fixed32(nil, 4, 1000)
	readings = fixed32(readings, 10, 4.5)

	var p []byte
	p = varint(p, 1, 3)
	p = fixed32(p, 28, 9999)
	p = fixed32(p, 30, 42)
	p = nest(p, 8, readings)

	h, err := DecodeHeartbeat(p)
	require.NoError(t, err)

	assert.Equal(t, LayoutNested, h.Layout)
	assert.InDelta(t, 1000, h.PowerW, 0.1)
	assert.InDelta(t, 4.5, h.Amps[0], 0.01)
}

// Field 17 is the session setpoint, field 18 the configured maximum: on an
// idle wallbox the setpoint may sit at 6 A whatever maximum is configured.
func TestDecodeHeartbeatReadsSetpointAndMaximum(t *testing.T) {
	var p []byte
	p = varint(p, 1, 1)
	p = varint(p, 17, 60)
	p = varint(p, 18, 160)

	h, err := DecodeHeartbeat(p)
	require.NoError(t, err)
	assert.Equal(t, uint64(60), h.ChargeCurrentRaw)
	assert.Equal(t, uint64(160), h.MaxCurrentRaw)
}

func TestDecodeHeartbeatPhaseMode(t *testing.T) {
	h, err := DecodeHeartbeat(varint(varint(nil, 1, 3), 21, 1))
	require.NoError(t, err)
	assert.True(t, h.HasPhaseMode)
	assert.Equal(t, uint64(1), h.PhaseModeRaw)

	h, err = DecodeHeartbeat(varint(nil, 1, 1))
	require.NoError(t, err)
	assert.False(t, h.HasPhaseMode, "phase mode reported although field 21 was absent")
}

// Real 2/33 pdata from the PowerPulse 2, idle, 2026-10-23 (gateway serial
// masked). It carries the nested field 8 record and top-level 29/30 side by side.
const capturedHeartbeat = "080112040000000030033a270800100025000000003d0000000045000000004d0000000055000000005d000000006500000000422a080110f4bde00125000000003d742a6243455e1568434db5d1664355b325993b5dde37d237651e7c9e3b48f4bde0018001048801a0019001a001a00100a80100b80103c50100004041c80164d80100e50100000000ea010c742a62435e156843b5d16643f2010cb325993bde37d2371e7c9e3bfa0102121482020200008a02020d0dc002fed2e8d606c80286a603d0020ad802eabde001e002f4bde001e80200f00208f80202ea030a08001000180020002800f203040800103cfa031a080110011a10484a333530303030303030303030303120012801800400a80600"

func TestDecodeCapturedHeartbeatPrefersNestedRecord(t *testing.T) {
	h, err := DecodeHeartbeat(mustHex(t, capturedHeartbeat))
	require.NoError(t, err)

	assert.Equal(t, LayoutNested, h.Layout)
	assert.Len(t, h.Volts, 3)
	assert.Len(t, h.Amps, 3)
	assert.Equal(t, StateUnplugged, h.State)
	assert.Equal(t, uint64(160), h.MaxCurrentRaw)
}

// reportPdata wraps a settings block in the 1 -> 4 -> 8 nesting of a 241/44 report.
func reportPdata(params []byte) []byte {
	return nest(nil, 1, nest(nil, 4, nest(nil, 8, params)))
}

func validSettings() []byte {
	var p []byte
	p = varint(p, 1, 0x12)       // switch bits
	p = varint(p, 2, 2)          // solar mode
	p = varint(p, 4, 160)        // 16.0 A max
	p = varint(p, 6, 60)         // 6.0 A solar minimum
	p = varint(p, 7, PhaseThree) // phase mode
	p = varint(p, 8, 100)        // 10.0 A custom
	return p
}

func TestDecodeSettings(t *testing.T) {
	s, err := DecodeSettings(reportPdata(validSettings()))
	require.NoError(t, err)
	assert.Equal(t, Settings{
		SwitchBits:     0x12,
		WorkMode:       2,
		MaxCurrentRaw:  160,
		SolarMinRaw:    60,
		PhaseModeRaw:   PhaseThree,
		UserCurrentRaw: 100,
	}, s)
}

// Every current field accepts up to 32 A, so a 20 A custom current does not
// drop the whole report.
func TestDecodeSettingsCurrentsUpTo32A(t *testing.T) {
	settings := func(user uint64) []byte {
		var p []byte
		for _, f := range [][2]uint64{{1, 0x12}, {2, 3}, {4, 320}, {6, 200}, {7, 2}, {8, user}} {
			p = varint(p, protowire.Number(f[0]), f[1])
		}
		return reportPdata(p)
	}

	s, err := DecodeSettings(settings(200))
	require.NoError(t, err)
	assert.Equal(t, uint64(320), s.MaxCurrentRaw)
	assert.Equal(t, uint64(200), s.SolarMinRaw)
	assert.Equal(t, uint64(200), s.UserCurrentRaw)

	_, err = DecodeSettings(settings(330))
	assert.ErrorContains(t, err, "field 8", "above 32 A")
}

// The gate stops a differently shaped 241/44 variant from being half-decoded
// into plausible-looking values.
func TestDecodeSettingsRejects(t *testing.T) {
	without := func(num protowire.Number) []byte {
		var p []byte
		for _, f := range [][2]uint64{{1, 0x12}, {2, 2}, {4, 160}, {6, 60}, {7, 2}, {8, 100}} {
			if protowire.Number(f[0]) != num {
				p = varint(p, protowire.Number(f[0]), f[1])
			}
		}
		return p
	}
	lvl8 := nest(nil, 8, validSettings())

	tests := map[string][]byte{
		"missing phase mode":      reportPdata(without(7)),
		"work mode out of range":  reportPdata(varint(without(2), 2, 9)),
		"max current below range": reportPdata(varint(without(4), 4, 10)),
		"no settings block":       nest(nil, 1, nest(nil, 4, varint(nil, 1, 1))),
		"duplicated nesting":      nest(nil, 1, append(nest(nil, 4, lvl8), nest(nil, 4, lvl8)...)),
	}

	for name, pdata := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeSettings(pdata)
			assert.Error(t, err)
		})
	}
}
