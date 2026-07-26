package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestImplausibleSessionEnergy(t *testing.T) {
	ctrl := gomock.NewController(t)
	rater := api.NewMockChargeRater(ctrl)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		chargeMeter: &Null{}, // silence nil panics
		chargeRater: rater,
		chargeTimer: &Null{}, // silence nil panics
	}

	// 2^32 Wh register garbage, see https://github.com/evcc-io/evcc/issues/32159
	rater.EXPECT().ChargedEnergy().Return(4294967.295, nil)
	lp.publishChargeProgress()
	assert.Zero(t, lp.energyMetrics.TotalWh(), "implausible reading not ignored")

	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	lp.publishChargeProgress()
	assert.Equal(t, 5e3, lp.energyMetrics.TotalWh(), "plausible reading not applied")
}
