package ecoflow

import (
	"testing"

	"github.com/evcc-io/evcc/charger/ecoflow/ecoflowtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedPdata unwraps the single command of a captured telemetry payload.
func capturedPdata(t *testing.T, payload []byte, fn, id uint64) []byte {
	t.Helper()
	headers, err := ParseMessage(payload)
	require.NoError(t, err)
	require.Len(t, headers, 1)
	gotFn, gotID, ok := headers[0].Command()
	require.True(t, ok)
	require.Equal(t, [2]uint64{fn, id}, [2]uint64{gotFn, gotID})
	pdata := headers[0].Pdata()
	require.Len(t, pdata, 1)
	return pdata[0]
}

// Every heartbeat the wallbox sent in the captures decodes to the values it
// showed at the time, so a parser change cannot silently shift a field.
func TestDecodeCapturedHeartbeats(t *testing.T) {
	tests := map[string]struct {
		state             State
		powerW            float32
		amps              [3]float32
		maxCurrent, phase uint64
		energyWh          uint64
	}{
		"state1-3p":       {StateUnplugged, 0, [3]float32{0, 0, 0}, 150, 0, 3677940},
		"state2-3p":       {StatePlugged, 0, [3]float32{0, 0, 0}, 150, 0, 3677940},
		"state2-1p":       {StatePlugged, 0, [3]float32{0, 0, 0}, 100, 1, 3678871},
		"state2-3p-fixed": {StatePlugged, 0, [3]float32{0, 0, 0}, 100, 0, 3692741},
		"state3-3p":       {StateCharging, 10857, [3]float32{15.92, 15.93, 16.04}, 160, 0, 3679185},
		"state3-3p-15a":   {StateCharging, 10254, [3]float32{15.08, 15.10, 15.16}, 150, 0, 3678066},
		"state3-1p":       {StateCharging, 2217, [3]float32{10.04, 0, 0}, 100, 1, 3678901},
		"state5-3p":       {StatePausedByVehicle, 7745, [3]float32{11.78, 11.82, 11.84}, 150, 0, 3677958},
		"state6-3p":       {StateStoppedByWallbox, 1, [3]float32{0.27, 0.27, 0.27}, 100, 0, 3678871},
		"state6-1p":       {StateStoppedByWallbox, 0, [3]float32{0.27, 0, 0}, 100, 1, 3692741},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h, err := DecodeHeartbeat(capturedPdata(t, ecoflowtest.Heartbeat(name), 2, 33))
			require.NoError(t, err)

			assert.Equal(t, LayoutNested, h.Layout)
			assert.Equal(t, tc.state, h.State)
			assert.InDelta(t, tc.powerW, h.PowerW, 1)
			require.Len(t, h.Amps, 3)
			require.Len(t, h.Volts, 3)
			for i := range 3 {
				assert.InDelta(t, tc.amps[i], h.Amps[i], 0.01, "amps L%d", i+1)
				assert.InDelta(t, 228, h.Volts[i], 8, "volts L%d", i+1)
			}
			assert.Equal(t, tc.maxCurrent, h.MaxCurrentRaw)
			assert.True(t, h.HasPhaseMode)
			assert.Equal(t, tc.phase, h.PhaseModeRaw)
			assert.Equal(t, tc.energyWh, h.TotalEnergyWh)
		})
	}
}

// Every full 241/44 report in the captures decodes completely; the short
// variant is rejected instead of half-decoded.
func TestDecodeCapturedSettings(t *testing.T) {
	report := func(maxCurrent, phase uint64) Settings {
		return Settings{SwitchBits: 3, WorkMode: 1, MaxCurrentRaw: maxCurrent, SolarMinRaw: 60, PhaseModeRaw: phase, UserCurrentRaw: 60}
	}
	tests := map[string]Settings{
		"max10":     report(100, PhaseAuto),
		"max15":     report(150, PhaseAuto),
		"max16":     report(160, PhaseAuto),
		"phase1p":   report(100, PhaseOne),
		"phase3p":   report(100, PhaseThree),
		"phaseauto": report(100, PhaseAuto),
	}

	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := DecodeSettings(capturedPdata(t, ecoflowtest.Settings(name), 241, 44))
			require.NoError(t, err)
			assert.Equal(t, want, s)
		})
	}

	_, err := DecodeSettings(capturedPdata(t, ecoflowtest.Settings("no-params"), 241, 44))
	assert.Error(t, err)
}
