package meter

// Conformance suite for EEBus MGCP TestSpec V1.0.1 ch.8 (Monitoring Appliance as DUT).
// evcc's grid meter is the MA; each ATC below maps to a subtest named by its ATC id.

import (
	"testing"

	eebusapi "github.com/enbility/eebus-go/api"
	mgcpmocks "github.com/enbility/eebus-go/usecases/mocks"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMGCPMeter wires an EEBus grid meter to a mocked MaMGCPInterface, as if a
// grid connection point had connected and announced its scenarios.
func newMGCPMeter(t *testing.T) (*EEBus, *mgcpmocks.MaMGCPInterface, spineapi.EntityRemoteInterface) {
	t.Helper()

	mm := mgcpmocks.NewMaMGCPInterface(t)
	entity := spinemocks.NewEntityRemoteInterface(t)

	c := &EEBus{
		log:       util.NewLogger("eebus-mgcp-test"),
		mm:        mm,
		maEntity:  entity,
		scenarios: mgcpScenarios,
	}

	return c, mm, entity
}

// nonNormalErrors model MGCP-TS-008: values in state "error"/"out of range" (or
// otherwise unusable) SHALL be ignored by the MA — evcc maps them to ErrNotAvailable.
var nonNormalErrors = []error{
	eebusapi.ErrDataInvalid,
	eebusapi.ErrDataNotAvailable,
	eebusapi.ErrMetadataNotAvailable,
}

// SCE2: total active power (ATC_SCE2_*_MATotalActivePower_*)
func TestMGCP_SCE2_TotalActivePower(t *testing.T) {
	// PT_001: state "normal"; MGCP-TS-010 consumption positive, production negative.
	t.Run("ATC_SCE2_PT_MATotalActivePower_001", func(t *testing.T) {
		for _, tc := range []struct {
			dir   string
			value float64
		}{
			{"consume", 4200},
			{"produce", -3100},
		} {
			t.Run(tc.dir, func(t *testing.T) {
				c, mm, entity := newMGCPMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPPower).Return(true)
				mm.EXPECT().Power(entity).Return(tc.value, nil)

				got, err := c.CurrentPower()
				require.NoError(t, err)
				assert.Equal(t, tc.value, got)
			})
		}
	})

	// NT_002: error/out-of-range value is discarded, never surfaced as a reading.
	t.Run("ATC_SCE2_NT_MATotalActivePower_002", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMGCPMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPPower).Return(true)
				mm.EXPECT().Power(entity).Return(0, badErr)

				_, err := c.CurrentPower()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})
}

// SCE4: total consumed energy (ATC_SCE4_*_MATotalConsumedEnergy_*)
func TestMGCP_SCE4_TotalConsumedEnergy(t *testing.T) {
	// PT_001: state "normal" while consuming; positive value per MGCP-TS-010.
	t.Run("ATC_SCE4_PT_MATotalConsumedEnergy_001", func(t *testing.T) {
		c, mm, entity := newMGCPMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPEnergyConsumed).Return(true)
		mm.EXPECT().EnergyConsumed(entity).Return(12345.6, nil)

		got, err := c.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, 12345.6, got)
	})

	// NT_002: error/out-of-range → discarded.
	t.Run("ATC_SCE4_NT_MATotalConsumedEnergy_002", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMGCPMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPEnergyConsumed).Return(true)
				mm.EXPECT().EnergyConsumed(entity).Return(0, badErr)

				_, err := c.TotalEnergy()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})
}

// SCE5: phase-specific AC current (ATC_SCE5_*_MAActiveACCurrent_*)
func TestMGCP_SCE5_ActiveACCurrent(t *testing.T) {
	// PT_001/003/005 (phase A/B/C, "normal"): evcc reads all phases in one call,
	// so a single Currents() covers the three per-phase positive cases.
	t.Run("ATC_SCE5_PT_MAActiveACCurrent_001_003_005", func(t *testing.T) {
		for _, tc := range []struct {
			dir      string
			a, b, cc float64
		}{
			{"consume", 6.1, 6.2, 6.3},
			{"produce", -6.1, -6.2, -6.3},
		} {
			t.Run(tc.dir, func(t *testing.T) {
				c, mm, entity := newMGCPMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPCurrentPerPhase).Return(true)
				mm.EXPECT().CurrentPerPhase(entity).Return([]float64{tc.a, tc.b, tc.cc}, nil)

				l1, l2, l3, err := c.Currents()
				require.NoError(t, err)
				assert.Equal(t, []float64{tc.a, tc.b, tc.cc}, []float64{l1, l2, l3})
			})
		}
	})

	// NT_002/004/006: error/out-of-range → discarded.
	t.Run("ATC_SCE5_NT_MAActiveACCurrent_002_004_006", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMGCPMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPCurrentPerPhase).Return(true)
				mm.EXPECT().CurrentPerPhase(entity).Return(nil, badErr)

				_, _, _, err := c.Currents()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})

	// MGCP-TS-006/7: only connected phases delivered; evcc pads to three phases.
	t.Run("partial_phases_padded", func(t *testing.T) {
		c, mm, entity := newMGCPMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPCurrentPerPhase).Return(true)
		mm.EXPECT().CurrentPerPhase(entity).Return([]float64{7.5}, nil)

		l1, l2, l3, err := c.Currents()
		require.NoError(t, err)
		assert.Equal(t, []float64{7.5, 0, 0}, []float64{l1, l2, l3})
	})

	// Malformed data (>3 phases) must not be surfaced as a reading.
	t.Run("too_many_phases_rejected", func(t *testing.T) {
		c, mm, entity := newMGCPMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPCurrentPerPhase).Return(true)
		mm.EXPECT().CurrentPerPhase(entity).Return([]float64{1, 2, 3, 4}, nil)

		_, _, _, err := c.Currents()
		assert.Error(t, err)
	})
}

// SCE6: phase-specific AC voltage (ATC_SCE6_*_MAACVoltage_*)
func TestMGCP_SCE6_ACVoltage(t *testing.T) {
	// PT_*: state "normal"; MGCP-TS-011 voltages independent of energy direction.
	t.Run("ATC_SCE6_PT_MAACVoltage", func(t *testing.T) {
		c, mm, entity := newMGCPMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPVoltagePerPhase).Return(true)
		mm.EXPECT().VoltagePerPhase(entity).Return([]float64{230.1, 229.8, 231.0}, nil)

		u1, u2, u3, err := c.Voltages()
		require.NoError(t, err)
		assert.Equal(t, []float64{230.1, 229.8, 231.0}, []float64{u1, u2, u3})
	})

	// NT_*: error/out-of-range → discarded.
	t.Run("ATC_SCE6_NT_MAACVoltage", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMGCPMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPVoltagePerPhase).Return(true)
				mm.EXPECT().VoltagePerPhase(entity).Return(nil, badErr)

				_, _, _, err := c.Voltages()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})
}

// Availability gating: an unannounced scenario or unconnected entity yields
// ErrNotAvailable — the MA must not invent a value for an unsupported data point.
func TestMGCP_ScenarioGating(t *testing.T) {
	t.Run("scenario_not_announced", func(t *testing.T) {
		c, mm, entity := newMGCPMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MGCPPower).Return(false)

		_, err := c.CurrentPower()
		assert.ErrorIs(t, err, api.ErrNotAvailable)
	})

	t.Run("entity_not_connected", func(t *testing.T) {
		c, _, _ := newMGCPMeter(t)
		c.maEntity = nil // GCP not (yet) connected

		_, err := c.CurrentPower()
		assert.ErrorIs(t, err, api.ErrNotAvailable)
	})
}

// MGCP-TS-009: the MA supports at least one of SCE2/3/4. evcc wires SCE2, SCE4
// plus SCE5/SCE6; these compile-time assertions guard the capabilities.
var (
	_ api.Meter         = (*EEBus)(nil)
	_ api.MeterEnergy   = (*EEBus)(nil)
	_ api.PhaseCurrents = (*EEBus)(nil)
	_ api.PhaseVoltages = (*EEBus)(nil)
)

// TestMGCPNonCoverage records the MA abstract test cases intentionally out of
// scope for evcc's grid meter, keeping the coverage map visible in test output.
func TestMGCPNonCoverage(t *testing.T) {
	for _, atc := range []string{
		"ATC_SCE1_PT_MAPowerLimitFactor_001",  // power-limit factor not exposed by api.Meter
		"ATC_SCE3_PT_MATotalFeedInEnergy_001", // feed-in energy: evcc reads consumed energy (SCE4) only
		"ATC_SCE3_NT_MATotalFeedInEnergy_002",
		"ATC_SCE7_PT_MAFrequency_001", // grid frequency not exposed by api.Meter
		"ATC_SCE7_NT_MAFrequency_002",
		"ATC_COM_PT_MAPolling_001",      // polling cadence owned by eebus-go
		"ATC_COM_PT_MANotification_001", // notification timing owned by eebus-go
	} {
		t.Run(atc, func(t *testing.T) {
			t.Skip("not applicable: evcc grid meter does not expose this MGCP data point")
		})
	}
}
