package eebus

import (
	"math"
	"testing"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitsValidate(t *testing.T) {
	require.NoError(t, Limits{}.validate())
	require.NoError(t, Limits{
		ContractualConsumptionNominalMax:    1,
		FailsafeConsumptionActivePowerLimit: 1,
		ProductionNominalMax:                1,
		FailsafeProductionActivePowerLimit:  new(0.0),
	}.validate())

	for name, l := range map[string]Limits{
		"contractual consumption": {ContractualConsumptionNominalMax: -1},
		"failsafe consumption":    {FailsafeConsumptionActivePowerLimit: -1},
		"production nominal":      {ProductionNominalMax: -1},
		"failsafe production":     {FailsafeProductionActivePowerLimit: new(-1.0)},
		"not a number":            {ProductionNominalMax: math.NaN()},
		"infinite":                {FailsafeConsumptionActivePowerLimit: math.Inf(1)},
	} {
		assert.Error(t, l.validate(), name)
	}
}

func TestRemoteLimitsWithInvalidSign(t *testing.T) {
	lpc := mocks.NewCsLPCInterface(t)
	lpp := mocks.NewCsLPPInterface(t)

	c := newTestEEBus(t)
	c.cs = &eebus.ControllableSystem{CsLPCInterface: lpc, CsLPPInterface: lpp}

	lpc.EXPECT().ConsumptionLimit().Return(ucapi.LoadLimit{Value: -1000, IsActive: true}, nil).Once()
	c.updateConsumptionLimit()
	assert.Zero(t, c.consumptionLimit)

	lpp.EXPECT().ProductionLimit().Return(ucapi.LoadLimit{Value: 1000, IsActive: true}, nil).Once()
	c.updateProductionLimit()
	assert.Zero(t, c.productionLimit)

	lpc.EXPECT().FailsafeConsumptionActivePowerLimit().Return(-1000, true, nil).Once()
	c.updateFailsafeConsumptionActivePowerLimit()
	assert.Equal(t, testFailsafeConsumption, c.failsafeConsumptionLimit)

	lpp.EXPECT().FailsafeProductionActivePowerLimit().Return(-1000, true, nil).Once()
	c.updateFailsafeProductionActivePowerLimit()
	assert.Equal(t, testFailsafeProduction, *c.failsafeProductionLimit)

	// valid signs are stored
	lpc.EXPECT().ConsumptionLimit().Return(ucapi.LoadLimit{Value: 1000, IsActive: true}, nil).Once()
	c.updateConsumptionLimit()
	assert.Equal(t, 1000.0, c.consumptionLimit.Value)

	lpp.EXPECT().ProductionLimit().Return(ucapi.LoadLimit{Value: -1000, IsActive: true}, nil).Once()
	c.updateProductionLimit()
	assert.Equal(t, -1000.0, c.productionLimit.Value)
}

func TestInvalidFailsafeLimit(t *testing.T) {
	config := func(key model.DeviceConfigurationKeyNameType, value float64) ucapi.PendingDeviceConfiguration {
		return ucapi.PendingDeviceConfiguration{
			KeyName: key,
			Value:   &model.DeviceConfigurationKeyValueValueType{ScaledNumber: model.NewScaledNumberType(value)},
		}
	}

	assert.True(t, invalidFailsafeLimit(config(model.DeviceConfigurationKeyNameTypeFailsafeConsumptionActivePowerLimit, -1000)))
	assert.True(t, invalidFailsafeLimit(config(model.DeviceConfigurationKeyNameTypeFailsafeProductionActivePowerLimit, -1000)))
	assert.False(t, invalidFailsafeLimit(config(model.DeviceConfigurationKeyNameTypeFailsafeProductionActivePowerLimit, 1000)))
	assert.False(t, invalidFailsafeLimit(config(model.DeviceConfigurationKeyNameTypeFailsafeDurationMinimum, -1)))
	assert.False(t, invalidFailsafeLimit(ucapi.PendingDeviceConfiguration{}))
}
