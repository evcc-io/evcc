package meter

// Conformance suite for EEBus LPC/LPP TestSpec V1.0.1 — Energy Guard (EG) role only.
// Grid meter = EG (Dim/SetCurtailPercent); Controllable System is the HEMS/charger, not the meter.

import (
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	eglpc "github.com/enbility/eebus-go/usecases/eg/lpc"
	eglpp "github.com/enbility/eebus-go/usecases/eg/lpp"
	egmocks "github.com/enbility/eebus-go/usecases/mocks"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newEGMeter(t *testing.T) (*EEBus, *egmocks.EgLPCInterface, *egmocks.EgLPPInterface, spineapi.EntityRemoteInterface) {
	t.Helper()

	lpc := egmocks.NewEgLPCInterface(t)
	lpp := egmocks.NewEgLPPInterface(t)
	entity := spinemocks.NewEntityRemoteInterface(t)

	c := &EEBus{
		ctx:         t.Context(),
		log:         util.NewLogger("eebus-eg-test"),
		eg:          &eebus.EnergyGuard{EgLPCInterface: lpc, EgLPPInterface: lpp},
		egLpcEntity: entity,
		egLppEntity: entity,
	}

	return c, lpc, lpp, entity
}

// ackWrite makes a mocked write invoke its result callback with a success result,
// so eebus.Await completes.
func ackWrite(_ spineapi.EntityRemoteInterface, _ ucapi.LoadLimit, cb func(model.ResultDataType, model.MsgCounterType)) {
	cb(model.ResultDataType{}, 0)
}

// captureWrite acks a mocked write and reports the limit it carried.
func captureWrite(written chan<- ucapi.LoadLimit) func(spineapi.EntityRemoteInterface, ucapi.LoadLimit, func(model.ResultDataType, model.MsgCounterType)) {
	return func(entity spineapi.EntityRemoteInterface, limit ucapi.LoadLimit, cb func(model.ResultDataType, model.MsgCounterType)) {
		written <- limit
		ackWrite(entity, limit, cb)
	}
}

// awaitLimit waits for a limit written by the background AssertLimit retry.
func awaitLimit(t *testing.T, written <-chan ucapi.LoadLimit) ucapi.LoadLimit {
	t.Helper()

	select {
	case limit := <-written:
		return limit
	case <-time.After(time.Second):
		t.Fatal("no limit written after connect")
		return ucapi.LoadLimit{}
	}
}

// --- LPC: Dim/Dimmed (Active Power Consumption Limit) -------------------------

// ATC_COM_PT_EGMessages_001/003 (LPC-TS-001/001-2): the EG sends an activated,
// then deactivated, consumption-limit write command. evcc's Dim writes a 0 W limit.
func TestLPC_EGMessages_ConsumptionLimit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dim    bool
		active bool
	}{
		{"activate", true, true},
		{"deactivate", false, false},
	} {
		t.Run("ATC_COM_PT_EGMessages_001_"+tc.name, func(t *testing.T) {
			c, lpc, _, entity := newEGMeter(t)
			lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true)
			lpc.EXPECT().
				WriteConsumptionLimit(entity, ucapi.LoadLimit{Value: 0, IsActive: tc.active}, mock.Anything).
				Run(ackWrite).
				Return(new(model.MsgCounterType), nil)

			assert.NoError(t, c.Dim(tc.dim))
		})
	}
}

// A rejected write (NACK) must surface as an error, not silent success.
func TestLPC_Dim_WriteRejected(t *testing.T) {
	c, lpc, _, entity := newEGMeter(t)
	lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true)
	lpc.EXPECT().
		WriteConsumptionLimit(entity, mock.Anything, mock.Anything).
		Run(func(_ spineapi.EntityRemoteInterface, _ ucapi.LoadLimit, cb func(model.ResultDataType, model.MsgCounterType)) {
			n := model.ErrorNumberType(7)
			cb(model.ResultDataType{ErrorNumber: &n}, 0)
		}).
		Return(new(model.MsgCounterType), nil)

	assert.Error(t, c.Dim(true))
}

// Dim is gated: no announced LPC scenario, or no connected entity → ErrNotAvailable.
func TestLPC_Dim_Gating(t *testing.T) {
	t.Run("scenario_not_announced", func(t *testing.T) {
		c, lpc, _, entity := newEGMeter(t)
		lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(false)

		assert.ErrorIs(t, c.Dim(true), api.ErrNotAvailable)
	})

	t.Run("entity_not_connected", func(t *testing.T) {
		c, _, _, _ := newEGMeter(t)
		c.egLpcEntity = nil

		assert.ErrorIs(t, c.Dim(true), api.ErrNotAvailable)
	})
}

// Dimmed reports an active consumption limit. Dim always writes a fixed 0W
// limit, so only IsActive determines the dimmed state (a value-based check
// would never report dimmed or release it).
func TestLPC_Dimmed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit ucapi.LoadLimit
		want  bool
	}{
		{"active_positive", ucapi.LoadLimit{IsActive: true, Value: 4000}, true},
		{"active_zero", ucapi.LoadLimit{IsActive: true, Value: 0}, true},
		{"inactive", ucapi.LoadLimit{IsActive: false, Value: 4000}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, lpc, _, entity := newEGMeter(t)
			lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true)
			lpc.EXPECT().ConsumptionLimit(entity).Return(tc.limit, nil)

			got, err := c.Dimmed()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- LPP: Curtail/Curtailed (Active Power Production Limit) -------------------

// ATC_COM_PT_EGMessages_001 (LPP-TS-001): the EG sends an activated/deactivated
// production-limit write command. LPP-TS-001 requires the value ≤ 0; evcc writes 0 W.
func TestLPP_EGMessages_ProductionLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		percent int
		active  bool
	}{
		{"activate", 0, true},
		{"deactivate", 100, false},
	} {
		t.Run("ATC_COM_PT_EGMessages_001_"+tc.name, func(t *testing.T) {
			c, _, lpp, entity := newEGMeter(t)
			lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true)
			if tc.active {
				lpp.EXPECT().ProductionNominalMax(entity).Return(0.0, api.ErrNotAvailable)
			}
			lpp.EXPECT().
				WriteProductionLimit(entity, ucapi.LoadLimit{Value: 0, IsActive: tc.active}, mock.Anything).
				Run(func(_ spineapi.EntityRemoteInterface, _ ucapi.LoadLimit, cb func(model.ResultDataType, model.MsgCounterType)) {
					cb(model.ResultDataType{}, 0)
				}).
				Return(new(model.MsgCounterType), nil)

			assert.NoError(t, c.SetCurtailPercent(tc.percent))
		})
	}
}

// A rejected write (NACK) must surface as an error, not silent success.
func TestLPP_Curtail_WriteRejected(t *testing.T) {
	c, _, lpp, entity := newEGMeter(t)
	lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true)
	lpp.EXPECT().ProductionNominalMax(entity).Return(0.0, api.ErrNotAvailable)
	lpp.EXPECT().
		WriteProductionLimit(entity, mock.Anything, mock.Anything).
		Run(func(_ spineapi.EntityRemoteInterface, _ ucapi.LoadLimit, cb func(model.ResultDataType, model.MsgCounterType)) {
			n := model.ErrorNumberType(7)
			cb(model.ResultDataType{ErrorNumber: &n}, 0)
		}).
		Return(new(model.MsgCounterType), nil)

	assert.Error(t, c.SetCurtailPercent(0))
}

// SetCurtailPercent is gated the same way as Dim.
func TestLPP_SetCurtailPercent_Gating(t *testing.T) {
	t.Run("scenario_not_announced", func(t *testing.T) {
		c, _, lpp, entity := newEGMeter(t)
		lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(false)

		assert.ErrorIs(t, c.SetCurtailPercent(0), api.ErrNotAvailable)
	})

	t.Run("entity_not_connected", func(t *testing.T) {
		c, _, _, _ := newEGMeter(t)
		c.egLppEntity = nil

		assert.ErrorIs(t, c.SetCurtailPercent(0), api.ErrNotAvailable)
	})
}

// CurtailedPercent expresses an active production limit as percent of nominal.
// Per LPP-TS-001 valid values are ≤ 0, so a positive value is not treated as curtailed.
func TestLPP_CurtailedPercent(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit ucapi.LoadLimit
		want  int
	}{
		{"active_negative", ucapi.LoadLimit{IsActive: true, Value: -2000}, 40},
		{"active_zero", ucapi.LoadLimit{IsActive: true, Value: 0}, 0},
		{"active_positive_invalid", ucapi.LoadLimit{IsActive: true, Value: 100}, 100},
		{"inactive", ucapi.LoadLimit{IsActive: false, Value: -2000}, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, lpp, entity := newEGMeter(t)
			lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true)
			lpp.EXPECT().ProductionLimit(entity).Return(tc.limit, nil)
			if tc.want != 100 {
				lpp.EXPECT().ProductionNominalMax(entity).Return(5000.0, nil)
			}

			got, err := c.CurtailedPercent()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// The watt conversion must reproduce the written percent, else the site would
// rewrite the same limit on every update.
func TestLPP_CurtailedPercent_RoundTrip(t *testing.T) {
	const nominal = 4600.0

	for percent := range 101 {
		c, _, lpp, entity := newEGMeter(t)
		lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true)
		lpp.EXPECT().ProductionLimit(entity).
			Return(ucapi.LoadLimit{IsActive: true, Value: -float64(percent) / 100 * nominal}, nil)
		lpp.EXPECT().ProductionNominalMax(entity).Return(nominal, nil)

		got, err := c.CurtailedPercent()
		require.NoError(t, err)
		assert.Equal(t, percent, got)
	}
}

// Without a nominal reference the watt limit cannot be expressed as a percent.
func TestLPP_CurtailedPercent_NoNominal(t *testing.T) {
	c, _, lpp, entity := newEGMeter(t)
	lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true)
	lpp.EXPECT().ProductionLimit(entity).Return(ucapi.LoadLimit{IsActive: true, Value: -2000}, nil)
	lpp.EXPECT().ProductionNominalMax(entity).Return(0.0, api.ErrNotAvailable)

	_, err := c.CurtailedPercent()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

// ATC_COM_PT_EGMessages_002 (LPC-913/LPP-913): after initial connection the EG
// states a limit to the CS - deactivated when nothing is being limited.
func TestLPC_LPP_InitialLimit(t *testing.T) {
	t.Run("consumption", func(t *testing.T) {
		c, lpc, _, entity := newEGMeter(t)
		c.egLpcEntity = nil

		written := make(chan ucapi.LoadLimit, 1)
		lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true)
		lpc.EXPECT().
			WriteConsumptionLimit(entity, mock.Anything, mock.Anything).
			Run(captureWrite(written)).
			Return(new(model.MsgCounterType), nil)

		c.UseCaseEvent(nil, entity, eglpc.UseCaseSupportUpdate)

		assert.Equal(t, ucapi.LoadLimit{Value: 0, IsActive: false}, awaitLimit(t, written))
	})

	t.Run("production", func(t *testing.T) {
		c, _, lpp, entity := newEGMeter(t)
		c.egLppEntity = nil
		c.curtailPercent = 100

		written := make(chan ucapi.LoadLimit, 1)
		lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true)
		lpp.EXPECT().
			WriteProductionLimit(entity, mock.Anything, mock.Anything).
			Run(captureWrite(written)).
			Return(new(model.MsgCounterType), nil)

		c.UseCaseEvent(nil, entity, eglpp.UseCaseSupportUpdate)

		assert.Equal(t, ucapi.LoadLimit{Value: 0, IsActive: false}, awaitLimit(t, written))
	})
}

// TestLPC_LPP_NonCoverage records the Controllable-System and connection/heartbeat
// abstract test cases that belong to eebus-go and the evcc HEMS/charger, not the meter.
func TestLPC_LPP_NonCoverage(t *testing.T) {
	for _, atc := range []string{
		"ATC_COM_PT_CSLimited_002",    // Controllable System role → charger/HEMS
		"ATC_COM_PT_CSFS_001",         // failsafe values → hems/eebus + eebus-go
		"ATC_COM_PT_EGHeartbeat_001",  // heartbeat cadence → eebus-go
		"ATC_COM_PT_EGConnection_001", // connection setup → eebus-go
	} {
		t.Run(atc, func(t *testing.T) {
			t.Skip("not applicable: covered by eebus-go or the evcc HEMS/charger, not the grid meter")
		})
	}
}
