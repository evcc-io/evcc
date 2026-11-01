package core

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

type batterySlaveTestController struct {
	modes []api.BatteryMode
}

func (c *batterySlaveTestController) BatteryModes() []api.BatteryMode {
	return c.modes
}

func (*batterySlaveTestController) SetBatteryMode(api.BatteryMode) error {
	return nil
}

type batterySlaveTestPowerController struct{}

func (*batterySlaveTestPowerController) SetBatteryModePower(api.BatteryMode, float64) error {
	return nil
}

func batterySlaveTestMeter(modes []api.BatteryMode, powerControl bool) api.Meter {
	controller := &batterySlaveTestController{modes: modes}
	if powerControl {
		return &struct {
			api.Meter
			api.BatteryController
			api.BatteryModePowerController
		}{
			BatteryController:          controller,
			BatteryModePowerController: &batterySlaveTestPowerController{},
		}
	}
	return &struct {
		api.Meter
		api.BatteryController
	}{BatteryController: controller}
}

func TestSupportsBatterySlaveControl(t *testing.T) {
	allModes := []api.BatteryMode{api.BatteryNormal, api.BatteryCharge, api.BatteryDischarge}
	chargeOnly := []api.BatteryMode{api.BatteryNormal, api.BatteryCharge}
	dischargeOnly := []api.BatteryMode{api.BatteryNormal, api.BatteryDischarge}

	tests := []struct {
		name  string
		meter api.Meter
		want  bool
	}{
		{name: "all required capabilities", meter: batterySlaveTestMeter(allModes, true), want: true},
		{name: "missing controller", meter: &struct{ api.Meter }{}},
		{name: "missing charge mode", meter: batterySlaveTestMeter(dischargeOnly, true)},
		{name: "missing discharge mode", meter: batterySlaveTestMeter(chargeOnly, true)},
		{name: "missing power setter", meter: batterySlaveTestMeter(allModes, false)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, SupportsBatterySlaveControl(test.meter))
		})
	}
}

func TestBatterySlaveModeAllowed(t *testing.T) {
	tests := []struct {
		name        string
		initialize  bool
		appliedMode api.BatteryMode
		want        bool
	}{
		{name: "nil cache", want: true},
		{name: "missing entry", initialize: true, want: true},
		{name: "unknown mode", initialize: true, appliedMode: api.BatteryUnknown, want: true},
		{name: "normal mode", initialize: true, appliedMode: api.BatteryNormal, want: true},
		{name: "charge mode", initialize: true, appliedMode: api.BatteryCharge},
		{name: "discharge mode", initialize: true, appliedMode: api.BatteryDischarge},
		{name: "hold mode", initialize: true, appliedMode: api.BatteryHold},
		{name: "hold charge mode", initialize: true, appliedMode: api.BatteryHoldCharge},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			site := &Site{}
			if tc.initialize {
				site.batteryModeApplied = map[string]api.BatteryMode{"slave": tc.appliedMode}
			}

			assert.Equal(t, tc.want, site.batterySlaveModeAllowed("slave"))
		})
	}
}

type batteryPowerControlMeter struct {
	power          float64
	readErr        error
	setErr         error
	modeSetErr     error
	modes          []api.BatteryMode
	modeSet        api.BatteryMode
	modeCalls      int
	modeSetHistory []api.BatteryMode
	mode           api.BatteryMode
	setpoint       float64
	setpointCalls  int
}

func (m *batteryPowerControlMeter) CurrentPower() (float64, error) {
	return m.power, m.readErr
}

func (m *batteryPowerControlMeter) SetBatteryModePower(mode api.BatteryMode, power float64) error {
	m.mode = mode
	m.setpoint = power
	m.setpointCalls++
	return m.setErr
}

func (m *batteryPowerControlMeter) BatteryModes() []api.BatteryMode {
	if m.modes != nil {
		return m.modes
	}
	return []api.BatteryMode{api.BatteryNormal, api.BatteryCharge, api.BatteryDischarge}
}

func (m *batteryPowerControlMeter) SetBatteryMode(mode api.BatteryMode) error {
	m.modeSet = mode
	m.modeCalls++
	m.modeSetHistory = append(m.modeSetHistory, mode)
	return m.modeSetErr
}

func TestUpdateBatterySlavePower(t *testing.T) {
	for _, tc := range []struct {
		name             string
		master           float64
		slaves           []float64
		slaveModes       []api.BatteryMode
		readErr          bool
		setErr           bool
		modeErr          bool
		mode             api.BatteryMode
		appliedMode      api.BatteryMode
		slaveAppliedMode api.BatteryMode
		wantSlaveMode    api.BatteryMode
		setpoint         float64
		wantCalls        int
		wantModeCalls    int
		wantStatus       string
	}{
		{name: "charge", master: -600, slaves: []float64{-300, -300}, mode: api.BatteryCharge, setpoint: 400, wantCalls: 1, wantModeCalls: 1, wantStatus: "applied", wantSlaveMode: api.BatteryCharge},
		{name: "discharge", master: 900, slaves: []float64{300, 0}, mode: api.BatteryDischarge, setpoint: 400, wantCalls: 1, wantModeCalls: 1, wantStatus: "applied", wantSlaveMode: api.BatteryDischarge},
		{name: "zero average", master: 0, slaves: []float64{0, 0}, mode: api.BatteryDischarge, setpoint: 0, wantCalls: 1, wantModeCalls: 1, wantStatus: "applied", wantSlaveMode: api.BatteryDischarge},
		{name: "incomplete readings", master: 900, slaves: []float64{300, 300}, readErr: true, wantStatus: "failed"},
		{name: "setter error", master: 600, slaves: []float64{300}, setErr: true, mode: api.BatteryDischarge, setpoint: 450, wantCalls: 1, wantModeCalls: 1, wantStatus: "failed", wantSlaveMode: api.BatteryDischarge},
		{name: "mode setter error", master: 600, slaves: []float64{300}, modeErr: true, mode: api.BatteryDischarge, setpoint: 450, wantModeCalls: 1, wantStatus: "failed"},
		{name: "unsupported slave mode", master: 600, slaves: []float64{300}, slaveModes: []api.BatteryMode{api.BatteryNormal, api.BatteryCharge}, wantStatus: "failed"},
		{name: "global normal mode", master: -600, slaves: []float64{-300}, mode: api.BatteryCharge, appliedMode: api.BatteryNormal, setpoint: 450, wantCalls: 1, wantModeCalls: 1, wantStatus: "applied", wantSlaveMode: api.BatteryCharge},
		{name: "global charge mode", master: -600, slaves: []float64{-300}, appliedMode: api.BatteryCharge, slaveAppliedMode: api.BatteryCharge, wantStatus: "overridden"},
		{name: "global hold mode", master: -600, slaves: []float64{-300}, appliedMode: api.BatteryHold, slaveAppliedMode: api.BatteryCharge, wantStatus: "overridden"},
		{name: "global override with zero average", master: 0, slaves: []float64{0}, appliedMode: api.BatteryCharge, slaveAppliedMode: api.BatteryCharge, wantStatus: "overridden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master := &batteryPowerControlMeter{power: tc.master}
			var masterMeter api.Meter = master
			meters := []config.Device[api.Meter]{config.NewStaticDevice(config.Named{Name: "master"}, masterMeter)}

			slaves := make([]*batteryPowerControlMeter, len(tc.slaves))
			for i, power := range tc.slaves {
				slaves[i] = &batteryPowerControlMeter{power: power}
				if tc.slaveModes != nil {
					slaves[i].modes = tc.slaveModes
				}
				if tc.setErr {
					slaves[i].setErr = errors.New("setpoint failed")
				}
				if tc.modeErr {
					slaves[i].modeSetErr = errors.New("mode failed")
				}
				if tc.readErr && i == len(tc.slaves)-1 {
					slaves[i].readErr = errors.New("power read failed")
				}

				var slaveMeter api.Meter = slaves[i]
				meters = append(meters, config.NewStaticDevice(config.Named{
					Name:  fmt.Sprintf("slave-%d", i+1),
					Other: map[string]any{"master": "master"},
				}, slaveMeter))
			}

			site := &Site{log: util.NewLogger("foo"), clock: clock.NewMock(), batteryMeters: meters}
			if tc.appliedMode != api.BatteryUnknown {
				site.batteryModeApplied = map[string]api.BatteryMode{"slave-1": tc.appliedMode}
			}
			if tc.slaveAppliedMode != api.BatteryUnknown {
				site.batterySlaveStates = map[string]*batterySlaveState{"slave-1": {appliedMode: tc.slaveAppliedMode}}
			}
			site.updateBatteryMeters()

			for i, slave := range slaves {
				assert.Equal(t, tc.wantCalls, slave.setpointCalls, "slave %d calls", i+1)
				assert.Equal(t, tc.wantModeCalls, slave.modeCalls, "slave %d mode calls", i+1)
				assert.Equal(t, tc.appliedMode, site.batteryModeApplied[fmt.Sprintf("slave-%d", i+1)], "slave %d global applied mode", i+1)
				state := site.batterySlaveStates[fmt.Sprintf("slave-%d", i+1)]
				appliedMode := api.BatteryUnknown
				if state != nil {
					appliedMode = state.appliedMode
				}
				assert.Equal(t, tc.wantSlaveMode, appliedMode, "slave %d resulting mode cache", i+1)
				assert.Equal(t, "master", site.battery.Devices[i+1].Master, "slave %d master", i+1)
				assert.Equal(t, tc.wantStatus, site.battery.Devices[i+1].PowerControlStatus, "slave %d power control status", i+1)
				if tc.wantModeCalls > 0 {
					assert.Equal(t, tc.mode, slave.modeSet, "slave %d controller mode", i+1)
				}
				if tc.wantCalls > 0 {
					assert.Equal(t, tc.mode, slave.mode, "slave %d mode", i+1)
					assert.Equal(t, tc.setpoint, slave.setpoint, "slave %d setpoint", i+1)
				}
			}

			expectedPower := tc.master
			for _, power := range tc.slaves {
				expectedPower += power
			}
			if tc.readErr {
				expectedPower -= tc.slaves[len(tc.slaves)-1]
			}
			assert.Equal(t, expectedPower, site.battery.Power, "measurements remain actual readings")
		})
	}
}

func TestUpdateBatterySlavePowerRejectsMismatchedInput(t *testing.T) {
	for _, tc := range []struct {
		name       string
		readings   int
		validFlags int
	}{
		{name: "missing measurement", readings: 1, validFlags: 2},
		{name: "missing validity flag", readings: 2, validFlags: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master := &batteryPowerControlMeter{}
			slave := &batteryPowerControlMeter{}
			var masterMeter api.Meter = master
			var slaveMeter api.Meter = slave
			site := &Site{
				log: util.NewLogger("foo"),
				batteryMeters: []config.Device[api.Meter]{
					config.NewStaticDevice(config.Named{Name: "master"}, masterMeter),
					config.NewStaticDevice(config.Named{Name: "slave", Other: map[string]any{"master": "master"}}, slaveMeter),
				},
			}

			site.updateBatterySlavePower(make([]types.Measurement, tc.readings), make([]bool, tc.validFlags))

			assert.Zero(t, slave.modeCalls)
			assert.Zero(t, slave.setpointCalls)
		})
	}
}

func TestUpdateBatterySlavePowerSkipsUnchangedMode(t *testing.T) {
	master := &batteryPowerControlMeter{power: -600}
	slave := &batteryPowerControlMeter{power: -300}
	var masterMeter api.Meter = master
	var slaveMeter api.Meter = slave
	site := &Site{
		log: util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{
			config.NewStaticDevice(config.Named{Name: "master"}, masterMeter),
			config.NewStaticDevice(config.Named{Name: "slave", Other: map[string]any{"master": "master"}}, slaveMeter),
		},
	}
	measurements := []types.Measurement{{Power: -600}, {Power: -300}}
	powerValid := []bool{true, true}

	site.updateBatterySlavePower(measurements, powerValid)
	site.updateBatterySlavePower(measurements, powerValid)

	assert.Equal(t, []api.BatteryMode{api.BatteryCharge}, slave.modeSetHistory)
	assert.Equal(t, 2, slave.setpointCalls)
	assert.Equal(t, api.BatteryCharge, site.batterySlaveStates["slave"].appliedMode)
}

func TestBatterySlaveSetpointMismatch(t *testing.T) {
	tests := []struct {
		name     string
		applied  batterySlaveSetpoint
		measured float64
		want     batterySlavePowerMismatch
	}{
		{
			name:     "charge while discharging",
			applied:  batterySlaveSetpoint{mode: api.BatteryCharge, power: 500},
			measured: 300,
			want:     batterySlavePowerOppositeDirection,
		},
		{
			name:     "discharge while charging",
			applied:  batterySlaveSetpoint{mode: api.BatteryDischarge, power: 500},
			measured: -300,
			want:     batterySlavePowerOppositeDirection,
		},
		{
			name:     "within charge deadband",
			applied:  batterySlaveSetpoint{mode: api.BatteryCharge, power: 500},
			measured: 50,
			want:     batterySlavePowerMatches,
		},
		{
			name:     "zero setpoint",
			applied:  batterySlaveSetpoint{mode: api.BatteryCharge, power: 0},
			measured: 300,
			want:     batterySlavePowerMatches,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, batterySlaveSetpointMismatch(tc.applied, tc.measured))
		})
	}
}

func TestUpdateBatterySlavePowerNewModeSkipsPreviousDirectionCheck(t *testing.T) {
	master := &batteryPowerControlMeter{power: -700}
	slave := &batteryPowerControlMeter{power: -300}
	var masterMeter api.Meter = master
	var slaveMeter api.Meter = slave
	site := &Site{
		log: util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{
			config.NewStaticDevice(config.Named{Name: "master"}, masterMeter),
			config.NewStaticDevice(config.Named{Name: "slave", Other: map[string]any{"master": "master"}}, slaveMeter),
		},
	}
	update := func() {
		measurements := []types.Measurement{{Power: master.power}, {Power: slave.power}}
		site.updateBatterySlavePower(measurements, []bool{true, true})
	}

	update()
	assert.Equal(t, api.BatteryCharge, slave.modeSet)

	master.power = 900
	slave.power = 300
	update()
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge, api.BatteryDischarge}, slave.modeSetHistory)
	assert.True(t, site.batterySlaveStates["slave"].directionMismatchSince.IsZero())
}

func TestUpdateBatterySlavePowerReappliesModeOnDirectionMismatch(t *testing.T) {
	master := &batteryPowerControlMeter{power: -700}
	slave := &batteryPowerControlMeter{power: -300}
	var masterMeter api.Meter = master
	var slaveMeter api.Meter = slave
	clck := clock.NewMock()
	site := &Site{
		log:   util.NewLogger("foo"),
		clock: clck,
		batteryMeters: []config.Device[api.Meter]{
			config.NewStaticDevice(config.Named{Name: "master"}, masterMeter),
			config.NewStaticDevice(config.Named{Name: "slave", Other: map[string]any{"master": "master"}}, slaveMeter),
		},
	}

	update := func() {
		measurements := []types.Measurement{{Power: master.power}, {Power: slave.power}}
		site.updateBatterySlavePower(measurements, []bool{true, true})
	}

	update()
	assert.Equal(t, api.BatteryCharge, slave.modeSet)
	assert.Equal(t, float64(500), slave.setpoint)
	slave.power = 300

	update()
	clck.Add(batterySlaveModeMismatchTimeout - time.Second)
	update()
	assert.Equal(t, 1, slave.modeCalls)

	clck.Add(time.Second)
	update()
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge, api.BatteryCharge}, slave.modeSetHistory)
	assert.Equal(t, float64(200), slave.setpoint)

	slave.power = -200
	update()
	assert.Len(t, slave.modeSetHistory, 2)
	assert.True(t, site.batterySlaveStates["slave"].directionMismatchSince.IsZero())
}

func TestUpdateBatterySlavePowerOutage(t *testing.T) {
	master := &batteryPowerControlMeter{}
	slave := &batteryPowerControlMeter{}
	var masterMeter api.Meter = master
	var slaveMeter api.Meter = slave
	clck := clock.NewMock()
	site := &Site{
		log:   util.NewLogger("foo"),
		clock: clck,
		batteryMeters: []config.Device[api.Meter]{
			config.NewStaticDevice(config.Named{Name: "master"}, masterMeter),
			config.NewStaticDevice(config.Named{Name: "slave", Other: map[string]any{"master": "master"}}, slaveMeter),
		},
	}

	update := func(slaveValid bool) types.Measurement {
		mm := []types.Measurement{{Power: -600}, {Power: -300}}
		site.updateBatterySlavePower(mm, []bool{true, slaveValid})
		return mm[1]
	}

	update(true)
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge}, slave.modeSetHistory)
	state := site.batterySlaveStates["slave"]
	state.directionMismatchSince = clck.Now()

	// outage within the timeout keeps the last setpoint
	assert.Equal(t, "failed", update(false).PowerControlStatus)
	assert.True(t, state.directionMismatchSince.IsZero())
	clck.Add(batterySlaveOutageTimeout - time.Second)
	update(false)
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge}, slave.modeSetHistory)

	// recovery restarts the timeout
	update(true)
	assert.Empty(t, site.batterySlaveOutageSince)
	update(false)
	clck.Add(batterySlaveOutageTimeout - time.Second)
	update(false)
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge}, slave.modeSetHistory)

	// timeout releases the slave once
	clck.Add(time.Second)
	update(false)
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge, api.BatteryNormal}, slave.modeSetHistory)
	assert.NotContains(t, site.batterySlaveStates, "slave")
	update(false)
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge, api.BatteryNormal}, slave.modeSetHistory)

	// readings return: control resumes
	update(true)
	assert.Equal(t, []api.BatteryMode{api.BatteryCharge, api.BatteryNormal, api.BatteryCharge}, slave.modeSetHistory)
	assert.Equal(t, api.BatteryCharge, site.batterySlaveStates["slave"].appliedMode)
}

func TestReleaseBatterySlaves(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cached        api.BatteryMode
		modeErr       error
		wantModeCalls int
		wantCached    bool
	}{
		{name: "charge", cached: api.BatteryCharge, wantModeCalls: 1},
		{name: "discharge", cached: api.BatteryDischarge, wantModeCalls: 1},
		{name: "normal", cached: api.BatteryNormal},
		{name: "never controlled"},
		{name: "mode error keeps cache", cached: api.BatteryCharge, modeErr: errors.New("mode failed"), wantModeCalls: 1, wantCached: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slave := &batteryPowerControlMeter{modeSetErr: tc.modeErr}
			var slaveMeter api.Meter = slave
			site := &Site{
				log: util.NewLogger("foo"),
				batteryMeters: []config.Device[api.Meter]{
					config.NewStaticDevice(config.Named{Name: "slave"}, slaveMeter),
				},
				batterySlaveStates: map[string]*batterySlaveState{},
			}
			if tc.cached != api.BatteryUnknown {
				site.batterySlaveStates["slave"] = &batterySlaveState{appliedMode: tc.cached}
			}

			site.releaseBatterySlaves()

			assert.Equal(t, tc.wantModeCalls, slave.modeCalls)
			if tc.wantModeCalls > 0 {
				assert.Equal(t, api.BatteryNormal, slave.modeSet)
			}
			_, cached := site.batterySlaveStates["slave"]
			assert.Equal(t, tc.wantCached || tc.cached == api.BatteryNormal, cached)
		})
	}
}

func TestApplyBatteryModeDoesNotClearSlaveModeCache(t *testing.T) {
	for _, tc := range []struct {
		name                string
		previousGlobalMode  api.BatteryMode
		requestedGlobalMode api.BatteryMode
		slaveMode           api.BatteryMode
		wantModeCalls       int
		wantSlaveCache      bool
	}{
		{
			name:                "unchanged normal leaves slave mode active",
			previousGlobalMode:  api.BatteryNormal,
			requestedGlobalMode: api.BatteryNormal,
			slaveMode:           api.BatteryCharge,
			wantSlaveCache:      true,
		},
		{
			name:                "active global mode transition overrides slave",
			previousGlobalMode:  api.BatteryNormal,
			requestedGlobalMode: api.BatteryCharge,
			slaveMode:           api.BatteryDischarge,
			wantModeCalls:       1,
			wantSlaveCache:      true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slave := &batteryPowerControlMeter{}
			var slaveMeter api.Meter = slave
			site := &Site{
				log: util.NewLogger("foo"),
				batteryMeters: []config.Device[api.Meter]{
					config.NewStaticDevice(config.Named{Name: "slave"}, slaveMeter),
				},
				batteryModeApplied: map[string]api.BatteryMode{"slave": tc.previousGlobalMode},
				batterySlaveStates: map[string]*batterySlaveState{
					"slave": {appliedMode: tc.slaveMode},
				},
			}

			assert.NoError(t, site.applyBatteryMode(tc.requestedGlobalMode))
			assert.Equal(t, tc.wantModeCalls, slave.modeCalls)
			_, slaveModeCached := site.batterySlaveStates["slave"]
			assert.Equal(t, tc.wantSlaveCache, slaveModeCached)
			if tc.wantModeCalls > 0 {
				assert.Equal(t, tc.requestedGlobalMode, slave.modeSet)
			}
		})
	}
}

func TestCalculateBatterySlaveSetpoints(t *testing.T) {
	tests := []struct {
		name        string
		masterPower float64
		slavePowers []float64
		wantValid   bool
		wantTotal   float64
		wantAverage float64
		wantMode    api.BatteryMode
		wantPower   float64
		wantCount   int
	}{
		{
			name:        "charge",
			masterPower: -600,
			slavePowers: []float64{-300, -300},
			wantValid:   true,
			wantTotal:   -1200,
			wantAverage: -400,
			wantMode:    api.BatteryCharge,
			wantPower:   400,
			wantCount:   2,
		},
		{
			name:        "discharge",
			masterPower: 900,
			slavePowers: []float64{300, 0},
			wantValid:   true,
			wantTotal:   1200,
			wantAverage: 400,
			wantMode:    api.BatteryDischarge,
			wantPower:   400,
			wantCount:   2,
		},
		{
			name:        "zero average",
			masterPower: 0,
			slavePowers: []float64{0, 0},
			wantValid:   true,
			wantMode:    api.BatteryDischarge,
			wantCount:   2,
		},
		{
			name:        "non-finite total",
			masterPower: 1e308,
			slavePowers: []float64{1e308},
		},
		{
			name: "no slaves",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, valid := calculateBatterySlaveSetpoints(tc.masterPower, tc.slavePowers)
			assert.Equal(t, tc.wantValid, valid)
			assert.Len(t, plan.setpoints, tc.wantCount)

			if !valid {
				return
			}

			assert.Equal(t, tc.wantTotal, plan.totalPower)
			assert.Equal(t, tc.wantAverage, plan.averagePower)
			for _, setpoint := range plan.setpoints {
				assert.Equal(t, tc.wantMode, setpoint.mode)
				assert.Equal(t, tc.wantPower, setpoint.power)
			}
		})
	}
}
