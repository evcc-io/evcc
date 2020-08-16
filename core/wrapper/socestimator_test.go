package wrapper

import (
	"testing"
	"time"

	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/util"
	"github.com/golang/mock/gomock"
)

func TestRemainingChargeDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := mock.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(int64(10))

	ce := NewSocEstimator(util.NewLogger("foo"), vehicle, false)
	ce.socCharge = 20.0

	chargePower := 1000.0
	targetSoC := 80

	if remaining := ce.RemainingChargeDuration(chargePower, targetSoC); remaining != 6*time.Hour {
		t.Error("wrong remaining charge duration")
	}
}
