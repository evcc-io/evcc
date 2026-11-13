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

	noMeter := func() loadpoint.API {
		lp := loadpoint.NewMockAPI(ctrl)
		lp.EXPECT().HasChargeMeter().Return(false)
		return lp
	}

	for _, tc := range []struct {
		name      string
		lp        loadpoint.API
		requested bool
		status    api.ChargeStatus
	}{
		{"no loadpoint, requested", nil, true, api.StatusC},
		{"no loadpoint, not requested", nil, false, api.StatusB},
		{"no meter, requested", noMeter(), true, api.StatusC},
		{"no meter, not requested", noMeter(), false, api.StatusB},
		{"standby, requested", withMeter(80), true, api.StatusB},
		{"running, requested", withMeter(800), true, api.StatusC},
		{"running, not requested", withMeter(800), false, api.StatusC},
	} {
		assert.Equal(t, tc.status, heatingStatus(tc.lp, heatpumpStandbyPower, tc.requested), tc.name)
	}
}
