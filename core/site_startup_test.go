package core

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestStartupOrder(t *testing.T) {
	ctrl := gomock.NewController(t)

	newLoadpoint := func(power float64, status api.ChargeStatus, err error) *Loadpoint {
		charger := api.NewMockCharger(ctrl)
		charger.EXPECT().Status().Return(status, err).AnyTimes()
		return &Loadpoint{charger: charger, chargePower: power}
	}

	var (
		idle1      = newLoadpoint(0, api.StatusA, nil)
		connected1 = newLoadpoint(0, api.StatusB, nil)
		charging1  = newLoadpoint(7000, api.StatusC, nil)
		standby    = newLoadpoint(standbyPower, api.StatusA, nil)
		failing    = newLoadpoint(0, api.StatusNone, errors.New("offline"))
		charging2  = newLoadpoint(11000, api.StatusB, nil) // charge power decides, not the status
		connected2 = newLoadpoint(5, api.StatusC, nil)
		faulted    = newLoadpoint(0, api.ChargeStatus("E"), nil)
	)

	got := startupOrder([]*Loadpoint{idle1, connected1, faulted, charging1, standby, failing, charging2, connected2})

	assert.Equal(t, []*Loadpoint{
		charging1, charging2, // charging
		connected1, connected2, // vehicle connected
		idle1, faulted, standby, failing, // remaining, order is kept within each group
	}, got)
}
