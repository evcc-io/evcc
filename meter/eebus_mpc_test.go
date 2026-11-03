package meter

// Conformance suite for EEBus MPC TestSpec V1.0.1 ch.8 (Monitoring Appliance as DUT).
// evcc's meter (non-grid usage) is the MA; it reads via MaMPCInterface.

import (
	"testing"

	mpcmocks "github.com/enbility/eebus-go/usecases/mocks"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMPCMeter(t *testing.T) (*EEBus, *mpcmocks.MaMPCInterface, spineapi.EntityRemoteInterface) {
	t.Helper()

	mm := mpcmocks.NewMaMPCInterface(t)
	entity := spinemocks.NewEntityRemoteInterface(t)

	c := &EEBus{
		log:       util.NewLogger("eebus-mpc-test"),
		mm:        mm,
		maEntity:  entity,
		scenarios: mpcScenarios,
	}

	return c, mm, entity
}

// SCE1: total active power (ATC_SCE1_*_MATotalActivePower_*)
func TestMPC_SCE1_TotalActivePower(t *testing.T) {
	// PT_001: state "normal"; MPC-TS-010 consumption positive, production negative.
	t.Run("ATC_SCE1_PT_MATotalActivePower_001", func(t *testing.T) {
		for _, tc := range []struct {
			dir   string
			value float64
		}{
			{"consume", 3300},
			{"produce", -1800},
		} {
			t.Run(tc.dir, func(t *testing.T) {
				c, mm, entity := newMPCMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCPower).Return(true)
				mm.EXPECT().Power(entity).Return(tc.value, nil)

				got, err := c.CurrentPower()
				require.NoError(t, err)
				assert.Equal(t, tc.value, got)
			})
		}
	})

	// NT_002: error/out-of-range → discarded (MPC-TS-008).
	t.Run("ATC_SCE1_NT_MATotalActivePower_002", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMPCMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCPower).Return(true)
				mm.EXPECT().Power(entity).Return(0, badErr)

				_, err := c.CurrentPower()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})
}

// SCE2: total consumed energy (ATC_SCE2_*_MATotalConsumedEnergy_*)
func TestMPC_SCE2_TotalConsumedEnergy(t *testing.T) {
	t.Run("ATC_SCE2_PT_MATotalConsumedEnergy_001", func(t *testing.T) {
		c, mm, entity := newMPCMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCEnergyConsumed).Return(true)
		mm.EXPECT().EnergyConsumed(entity).Return(9876.5, nil)

		got, err := c.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, 9876.5, got)
	})

	t.Run("ATC_SCE2_NT_MATotalConsumedEnergy_002", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMPCMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCEnergyConsumed).Return(true)
				mm.EXPECT().EnergyConsumed(entity).Return(0, badErr)

				_, err := c.TotalEnergy()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})
}

// SCE3: phase-specific AC current (ATC_SCE3_*_MAActiveACCurrent_*)
func TestMPC_SCE3_ActiveACCurrent(t *testing.T) {
	// PT_001/003/005 (phase A/B/C, "normal") in one Currents() call.
	t.Run("ATC_SCE3_PT_MAActiveACCurrent_001_003_005", func(t *testing.T) {
		c, mm, entity := newMPCMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCCurrentPerPhase).Return(true)
		mm.EXPECT().CurrentPerPhase(entity).Return([]float64{5.1, 5.2, 5.3}, nil)

		l1, l2, l3, err := c.Currents()
		require.NoError(t, err)
		assert.Equal(t, []float64{5.1, 5.2, 5.3}, []float64{l1, l2, l3})
	})

	// NT_002/004/006: error/out-of-range → discarded.
	t.Run("ATC_SCE3_NT_MAActiveACCurrent_002_004_006", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMPCMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCCurrentPerPhase).Return(true)
				mm.EXPECT().CurrentPerPhase(entity).Return(nil, badErr)

				_, _, _, err := c.Currents()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})
}

// SCE4: phase-specific AC voltage (ATC_SCE4_*_MAACVoltage_*)
func TestMPC_SCE4_ACVoltage(t *testing.T) {
	t.Run("ATC_SCE4_PT_MAACVoltage", func(t *testing.T) {
		c, mm, entity := newMPCMeter(t)
		mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCVoltagePerPhase).Return(true)
		mm.EXPECT().VoltagePerPhase(entity).Return([]float64{230.0, 230.5, 229.5}, nil)

		u1, u2, u3, err := c.Voltages()
		require.NoError(t, err)
		assert.Equal(t, []float64{230.0, 230.5, 229.5}, []float64{u1, u2, u3})
	})

	t.Run("ATC_SCE4_NT_MAACVoltage", func(t *testing.T) {
		for _, badErr := range nonNormalErrors {
			t.Run(badErr.Error(), func(t *testing.T) {
				c, mm, entity := newMPCMeter(t)
				mm.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.MPCVoltagePerPhase).Return(true)
				mm.EXPECT().VoltagePerPhase(entity).Return(nil, badErr)

				_, _, _, err := c.Voltages()
				assert.ErrorIs(t, err, api.ErrNotAvailable)
			})
		}
	})
}

// TestMPCNonCoverage records MPC MA abstract test cases out of scope for evcc.
func TestMPCNonCoverage(t *testing.T) {
	for _, atc := range []string{
		"ATC_SCE1_PT_MAPhaseActivePower_001",    // per-phase active power not exposed by api.Meter
		"ATC_SCE2_PT_MATotalProducedEnergy_001", // produced energy not exposed (consumed only)
		"ATC_SCE5_PT_MAFrequency_001",           // grid frequency not exposed by api.Meter
		"ATC_COM_PT_MAPolling_001",              // polling cadence owned by eebus-go
		"ATC_COM_PT_MANotification_001",         // notification timing owned by eebus-go
	} {
		t.Run(atc, func(t *testing.T) {
			t.Skip("not applicable: evcc meter does not expose this MPC data point")
		})
	}
}
