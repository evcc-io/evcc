package charger

// Conformance suite for EEBus LPC TestSpec V1.0.1 — Energy Guard (EG) role.
// The OHPCF charger is the EG via Dim/Dimmed; it has no LPP (a heat pump only consumes).

import (
	"testing"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
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

func newOHPCFEGCharger(t *testing.T) (*EEBusOHPCF, *egmocks.EgLPCInterface, spineapi.EntityRemoteInterface) {
	t.Helper()

	lpc := egmocks.NewEgLPCInterface(t)
	entity := spinemocks.NewEntityRemoteInterface(t)

	c := &EEBusOHPCF{
		log:         util.NewLogger("eebus-ohpcf-test"),
		eg:          &eebus.EnergyGuard{EgLPCInterface: lpc},
		egLpcEntity: entity,
	}

	return c, lpc, entity
}

// ATC_COM_PT_EGMessages_001/003 (LPC-TS-001/001-2): the EG sends an activated,
// then deactivated, consumption-limit write command. Dim writes a 0 W limit.
func TestOHPCF_LPC_EGMessages_ConsumptionLimit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dim    bool
		active bool
	}{
		{"activate", true, true},
		{"deactivate", false, false},
	} {
		t.Run("ATC_COM_PT_EGMessages_001_"+tc.name, func(t *testing.T) {
			c, lpc, entity := newOHPCFEGCharger(t)
			lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true)
			lpc.EXPECT().
				WriteConsumptionLimit(entity, ucapi.LoadLimit{Value: 0, IsActive: tc.active}, mock.Anything).
				Run(func(_ spineapi.EntityRemoteInterface, _ ucapi.LoadLimit, cb func(model.ResultDataType, model.MsgCounterType)) {
					cb(model.ResultDataType{}, 0)
				}).
				Return(new(model.MsgCounterType), nil)

			assert.NoError(t, c.Dim(tc.dim))
		})
	}
}

// A rejected write (NACK) must surface as an error, not silent success.
func TestOHPCF_LPC_Dim_WriteRejected(t *testing.T) {
	c, lpc, entity := newOHPCFEGCharger(t)
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
func TestOHPCF_LPC_Dim_Gating(t *testing.T) {
	t.Run("scenario_not_announced", func(t *testing.T) {
		c, lpc, entity := newOHPCFEGCharger(t)
		lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(false)

		assert.ErrorIs(t, c.Dim(true), api.ErrNotAvailable)
	})

	t.Run("entity_not_connected", func(t *testing.T) {
		c, _, _ := newOHPCFEGCharger(t)
		c.egLpcEntity = nil

		assert.ErrorIs(t, c.Dim(true), api.ErrNotAvailable)
	})
}

// Dimmed reports an active consumption limit. Dim always writes a fixed 0W
// limit, so only IsActive determines the dimmed state (a value-based check
// would never report dimmed or release it).
func TestOHPCF_LPC_Dimmed(t *testing.T) {
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
			c, lpc, entity := newOHPCFEGCharger(t)
			lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true)
			lpc.EXPECT().ConsumptionLimit(entity).Return(tc.limit, nil)

			got, err := c.Dimmed()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Dimmed is gated like Dim: no announced LPC scenario, or no connected entity → ErrNotAvailable.
func TestOHPCF_LPC_Dimmed_Gating(t *testing.T) {
	t.Run("scenario_not_announced", func(t *testing.T) {
		c, lpc, entity := newOHPCFEGCharger(t)
		lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(false)

		_, err := c.Dimmed()
		assert.ErrorIs(t, err, api.ErrNotAvailable)
	})

	t.Run("entity_not_connected", func(t *testing.T) {
		c, _, _ := newOHPCFEGCharger(t)
		c.egLpcEntity = nil

		_, err := c.Dimmed()
		assert.ErrorIs(t, err, api.ErrNotAvailable)
	})
}

// Dimmed discards a non-normal limit value (LPC-TS-008) as ErrNotAvailable.
func TestOHPCF_LPC_Dimmed_Discard(t *testing.T) {
	for _, badErr := range []error{eebusapi.ErrDataInvalid, eebusapi.ErrDataNotAvailable, eebusapi.ErrMetadataNotAvailable} {
		t.Run(badErr.Error(), func(t *testing.T) {
			c, lpc, entity := newOHPCFEGCharger(t)
			lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true)
			lpc.EXPECT().ConsumptionLimit(entity).Return(ucapi.LoadLimit{}, badErr)

			_, err := c.Dimmed()
			assert.ErrorIs(t, err, api.ErrNotAvailable)
		})
	}
}
