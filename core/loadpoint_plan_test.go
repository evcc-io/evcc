package core

import (
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPlanRequiredDurationVehicleRemoved(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.vehicleSoc = 50

	// remove the vehicle while socBasedPlanning reads its capacity, i.e. before the snapshot
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().DoAndReturn(func() float64 {
		lp.vmu.Lock()
		lp.vehicle = nil
		lp.vmu.Unlock()
		return 50
	})
	lp.vehicle = vehicle

	assert.Equal(t, time.Duration(0), lp.GetPlanRequiredDuration(80, 10e3))
}

func newUnsampledSocLoadpoint(t *testing.T) *Loadpoint {
	ctrl := gomock.NewController(t)

	// 8.5 kWh capacity => 10 kWh virtual capacity at 85% efficiency
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(8.5).AnyTimes()
	vehicle.EXPECT().Features().AnyTimes()

	log := util.NewLogger("foo")
	return &Loadpoint{
		log:          log,
		vehicle:      vehicle,
		status:       api.StatusC,
		vehicleSoc:   40, // stale SoC from a previous session
		socEstimator: soc.NewEstimator(log, vehicle),
	}
}

func TestPlanRequiredDurationUnsampledSoc(t *testing.T) {
	lp := newUnsampledSocLoadpoint(t)

	for _, tc := range []struct {
		charged  float64 // kWh
		expected time.Duration
	}{
		{0, 3*time.Hour + 30*time.Minute},
		{2, time.Hour + 30*time.Minute},
		{3.5, 0},
		{25, 0},
	} {
		lp.energyMetrics.totalKWh = tc.charged
		assert.Equal(t, tc.expected, lp.getPlanRequiredDuration(35, 1000), "charged %.1fkWh", tc.charged)
	}

	// energy of a previous session doesn't count while disconnected
	lp.status = api.StatusA
	assert.Equal(t, 3*time.Hour+30*time.Minute, lp.getPlanRequiredDuration(35, 1000))

	// a valid SoC takes over
	lp.status = api.StatusC
	lp.energyMetrics.totalKWh = 25
	vehicleSoc := 30.0
	lp.socEstimator.Soc(&vehicleSoc, lp.getChargedEnergy())
	assert.Equal(t, 30*time.Minute, lp.getPlanRequiredDuration(35, 1000))
}

func TestNextActivePlanUnsampledSoc(t *testing.T) {
	lp := newUnsampledSocLoadpoint(t)
	plans := []plan{{Id: 2, Soc: 35, End: time.Now().Add(12 * time.Hour)}}

	// stale SoC above the goal must not skip the plan
	p := lp.nextActivePlan(1000, plans)
	require.NotNil(t, p)
	assert.Equal(t, 2, p.Id)

	// goal energy charged
	lp.energyMetrics.totalKWh = 3.5
	assert.Nil(t, lp.nextActivePlan(1000, plans))
}

func TestPlanActiveRace(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			lp.setPlanActive(i%2 == 0)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			_ = lp.IsFastChargingActive()
		}
	}()
	wg.Wait()
}

func TestPlanLockConcurrentAccess(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	goalTime := time.Now()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, action := range []func(){
		func() { lp.lockPlanGoal(goalTime, 80, 2) },
		func() { lp.ClearPlanLock() },
		func() { lp.setPlanActive(false) },
		func() {
			when, soc, id := lp.nextVehiclePlan()
			if id != 0 {
				assert.Equal(t, goalTime, when)
				assert.Equal(t, 80, soc)
				assert.Equal(t, 2, id)
			}
		},
	} {
		wg.Go(func() {
			<-start
			for range 1000 {
				action()
			}
		})
	}
	close(start)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent plan operations did not finish")
	}
}

func TestPlanLockWithLoadpointMutex(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*Loadpoint)
	}{
		{"clear", func(lp *Loadpoint) { lp.clearPlanLock() }},
		{"reset", func(lp *Loadpoint) { lp.setPlanActive(false) }},
		{"lock", func(lp *Loadpoint) { lp.lockPlanGoal(time.Now(), 80, 2) }},
		{"read", func(lp *Loadpoint) { lp.nextVehiclePlan() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lp := NewLoadpoint(util.NewLogger("foo"), nil)
			lp.lockPlanGoal(time.Now(), 80, 2)
			done := make(chan struct{})
			go func() {
				lp.Lock()
				defer lp.Unlock()
				tc.run(lp)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("plan operation deadlocked with the loadpoint mutex held")
			}
		})
	}
}
