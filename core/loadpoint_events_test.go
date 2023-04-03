package core

import (
	"fmt"
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type pushMessenger struct {
}

func (p *pushMessenger) Send(title, msg string) {
	fmt.Println(title, msg)
	panic("SENT")
}

func TestDisconnect(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := mock.NewMockCharger(ctrl)
	vehicle := mock.NewMockVehicle(ctrl)

	// wrap vehicle with estimator
	vehicle.EXPECT().Capacity().Return(float64(10))
	vehicle.EXPECT().Phases().Return(0).AnyTimes()
	socEstimator := soc.NewEstimator(util.NewLogger("foo"), charger, vehicle, false)

	valueChan := make(chan util.Param, 1)
	cache := util.NewCache()
	go cache.Run(valueChan)

	pushChan := make(chan push.Event, 1)
	hub, err := push.NewHub(map[string]push.EventTemplateConfig{
		evVehicleDisconnect: {
			Title: "foo",
			Msg:   "{{.vehicleTitle}} disconnected",
		},
	}, cache)
	require.NoError(t, err)

	msgr := &pushMessenger{}
	hub.Add(msgr)

	go hub.Run(pushChan, valueChan)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		pushChan:    pushChan,
		charger:     charger,
		chargeMeter: &Null{},            // silence nil panics
		chargeRater: &Null{},            // silence nil panics
		chargeTimer: &Null{},            // silence nil panics
		progress:    NewProgress(0, 10), // silence nil panics
		wakeUpTimer: NewTimer(),         // silence nil panics
		// coordinator:  coordinator.NewDummy(), // silence nil panics
		MinCurrent:   minA,
		MaxCurrent:   maxA,
		vehicle:      vehicle,      // needed for targetSoc check
		socEstimator: socEstimator, // instead of vehicle: vehicle,
		Mode:         api.ModeNow,
	}

	valueChan <- util.Param{
		Key: "vehicleTitle",
		Val: "Dudu",
	}
	lp.pushEvent(evVehicleDisconnect)

	// time.Sleep(1 * time.Second)
}
