package core

import (
	"errors"
	"fmt"
	"testing"

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
		name          string
		master        float64
		slaves        []float64
		slaveModes    []api.BatteryMode
		readErr       bool
		setErr        bool
		modeErr       bool
		mode          api.BatteryMode
		wantSlaveMode api.BatteryMode
		setpoint      float64
		wantCalls     int
		wantModeCalls int
		wantStatus    string
	}{
		{name: "charge", master: -600, slaves: []float64{-300, -300}, mode: api.BatteryCharge, wantSlaveMode: api.BatteryCharge, setpoint: 400, wantCalls: 1, wantModeCalls: 1, wantStatus: "applied"},
		{name: "discharge", master: 900, slaves: []float64{300, 0}, mode: api.BatteryDischarge, wantSlaveMode: api.BatteryDischarge, setpoint: 400, wantCalls: 1, wantModeCalls: 1, wantStatus: "applied"},
		{name: "zero average", master: 0, slaves: []float64{0, 0}, mode: api.BatteryDischarge, wantSlaveMode: api.BatteryDischarge, setpoint: 0, wantCalls: 1, wantModeCalls: 1, wantStatus: "applied"},
		{name: "incomplete readings", master: 900, slaves: []float64{300, 300}, readErr: true, wantStatus: "failed"},
		{name: "setter error", master: 600, slaves: []float64{300}, setErr: true, mode: api.BatteryDischarge, wantSlaveMode: api.BatteryDischarge, setpoint: 450, wantCalls: 1, wantModeCalls: 1, wantStatus: "failed"},
		{name: "mode setter error", master: 600, slaves: []float64{300}, modeErr: true, mode: api.BatteryDischarge, setpoint: 450, wantModeCalls: 1, wantStatus: "failed"},
		{name: "unsupported slave mode", master: 600, slaves: []float64{300}, slaveModes: []api.BatteryMode{api.BatteryNormal, api.BatteryCharge}, wantStatus: "failed"},
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

			site := &Site{log: util.NewLogger("foo"), batteryMeters: meters}
			site.updateBatteryMeters()

			for i, slave := range slaves {
				assert.Equal(t, tc.wantCalls, slave.setpointCalls, "slave %d calls", i+1)
				assert.Equal(t, tc.wantModeCalls, slave.modeCalls, "slave %d mode calls", i+1)
				assert.Equal(t, tc.wantSlaveMode, site.batterySlaveModeApplied[fmt.Sprintf("slave-%d", i+1)], "slave %d slave mode cache", i+1)
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
	assert.Equal(t, api.BatteryCharge, site.batterySlaveModeApplied["slave"])
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
				batterySlaveModeApplied: map[string]api.BatteryMode{},
			}
			if tc.cached != api.BatteryUnknown {
				site.batterySlaveModeApplied["slave"] = tc.cached
			}

			site.releaseBatterySlaves()

			assert.Equal(t, tc.wantModeCalls, slave.modeCalls)
			if tc.wantModeCalls > 0 {
				assert.Equal(t, api.BatteryNormal, slave.modeSet)
			}
			_, cached := site.batterySlaveModeApplied["slave"]
			assert.Equal(t, tc.wantCached || tc.cached == api.BatteryNormal, cached)
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
