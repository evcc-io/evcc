package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHeatingStatus(t *testing.T) {
	ctrl := gomock.NewController(t)

	withMeter := func(power float64) loadpoint.API {
		lp := loadpoint.NewMockAPI(ctrl)
		lp.EXPECT().HasChargeMeter().Return(true)
		lp.EXPECT().GetChargePower().Return(power)
		return lp
	}

	noMeter := loadpoint.NewMockAPI(ctrl)
	noMeter.EXPECT().HasChargeMeter().Return(false)

	for _, tc := range []struct {
		name   string
		lp     loadpoint.API
		status api.ChargeStatus
	}{
		{"no loadpoint", nil, api.StatusC},
		{"no meter", noMeter, api.StatusC},
		{"standby", withMeter(80), api.StatusB},
		{"running", withMeter(800), api.StatusC},
	} {
		assert.Equal(t, tc.status, heatingStatus(tc.lp, heatpumpStandbyPower), tc.name)
	}
}
