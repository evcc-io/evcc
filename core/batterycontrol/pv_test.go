package batterycontrol

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

type fakeDevice struct {
	now                           time.Time
	state                         api.BatteryControlState
	calls                         []string
	targets                       []float64
	readErr, writeErr, releaseErr error
	stale, unacknowledged         bool
	previous                      string
}

func newDevice() *fakeDevice {
	return &fakeDevice{now: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC), state: api.BatteryControlState{Mode: api.BatteryOperatingAuto, NativeMode: "10", Soc: 50}}
}
func (d *fakeDevice) BatteryControlState() (api.BatteryControlState, error) {
	if d.state.LeaseID != "" && !d.state.LeaseUntil.After(d.now) {
		d.state.Mode = api.BatteryOperatingAuto
		d.state.NativeMode = d.previous
		d.state.Power = 0
		d.state.LeaseID = ""
	}
	s := d.state
	s.ObservedAt = d.now
	if d.stale {
		s.ObservedAt = d.now.Add(-MaxSampleAge - time.Second)
	}
	return s, d.readErr
}
func (d *fakeDevice) AcquireBatteryPVControl(id string, power float64, ttl time.Duration) error {
	d.calls = append(d.calls, "acquire")
	if d.state.Mode != api.BatteryOperatingAuto || d.state.Power < -5 || d.state.NativeCharging || d.state.LeaseID != "" {
		return errors.New("compare failed")
	}
	d.previous = d.state.NativeMode
	d.state.Mode = api.BatteryOperatingManual
	d.state.NativeMode = "1"
	d.state.LeaseID = id
	d.state.LeaseUntil = d.now.Add(ttl)
	d.targets = append(d.targets, power)
	if d.unacknowledged {
		d.state.LeaseID = ""
	}
	return d.writeErr
}
func (d *fakeDevice) RenewBatteryPVControl(id string, power float64, ttl time.Duration) error {
	d.calls = append(d.calls, "renew")
	if id != d.state.LeaseID || !d.state.LeaseUntil.After(d.now) {
		return errors.New("lease lost")
	}
	d.state.LeaseUntil = d.now.Add(ttl)
	d.targets = append(d.targets, power)
	return d.writeErr
}
func (d *fakeDevice) ReleaseBatteryPVControl(id string) error {
	d.calls = append(d.calls, "release")
	if d.releaseErr != nil {
		return d.releaseErr
	}
	if id == d.state.LeaseID {
		d.state.Mode = api.BatteryOperatingAuto
		d.state.NativeMode = d.previous
		d.state.LeaseID = ""
		d.state.Power = 0
	}
	return nil
}
func inputs() Input {
	return Input{Valid: true, Allowed: true, Grid: -500, PV: 5000, Reserve: 100, StartPower: 500, MaxPower: 3300, MaxSoc: 100}
}
func session(d *fakeDevice) *Session { return &Session{now: func() time.Time { return d.now }} }

func TestPVNeverClaimsNativeOrUnknown(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mode     api.BatteryOperatingMode
		power    float64
		charging bool
		lease    string
		want     Phase
	}{
		{"native charging", api.BatteryOperatingAuto, -900, false, "", Native},
		{"native tiny charging with flag", api.BatteryOperatingAuto, -3, true, "", Native},
		{"native charging flag", api.BatteryOperatingAuto, 0, true, "", Native},
		{"manual startup", api.BatteryOperatingManual, 0, false, "", Foreign},
		{"previous process lease", api.BatteryOperatingManual, -500, false, "old-process", Foreign},
		{"unknown mode", api.BatteryOperatingUnknown, 0, false, "", Foreign},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newDevice()
			d.state.Mode = tc.mode
			d.state.Power = tc.power
			d.state.NativeCharging = tc.charging
			d.state.LeaseID = tc.lease
			d.state.LeaseUntil = d.now.Add(time.Hour)
			s := session(d)
			in := inputs()
			in.Grid = -3000
			for range 5 {
				require.NoError(t, s.Step(d, d, in))
			}
			require.Equal(t, tc.want, s.Phase)
			require.Empty(t, d.calls)
		})
	}
}
func TestPVStartContinueBalanceImportStop(t *testing.T) {
	d := newDevice()
	s := session(d)
	in := inputs()
	in.Grid = -499
	require.NoError(t, s.Step(d, d, in))
	require.Empty(t, d.calls)
	in.Grid = -500
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, Active, s.Phase)
	require.Equal(t, 400.0, s.Power())
	d.state.Power = -400
	in.Grid = 0
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, 400.0, s.Power())
	in.Grid = -100
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, 400.0, s.Power())
	in.Grid = -200
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, 500.0, s.Power())
	d.state.Power = -500
	in.Grid = 1
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, Observing, s.Phase)
	require.Equal(t, "10", d.state.NativeMode)
	in.Grid = -499
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, Observing, s.Phase)
}

func TestPVStartThresholdIndependentOfChargeLimit(t *testing.T) {
	for _, limit := range []float64{200, 3300, 3500, 10000} {
		d := newDevice()
		s := session(d)
		in := inputs()
		in.MaxPower = limit
		in.Grid = -499
		require.NoError(t, s.Step(d, d, in))
		require.Empty(t, d.calls)
		in.Grid = -500
		require.NoError(t, s.Step(d, d, in))
		require.Equal(t, min(400.0, limit), s.Power())
	}
}

func TestPVStartUsesConfiguredValueOnlyBeforeCharging(t *testing.T) {
	for _, start := range []float64{250, 500, 750, 1500} {
		d := newDevice()
		s := session(d)
		in := inputs()
		in.StartPower = start
		in.Grid = -start + 1
		require.NoError(t, s.Step(d, d, in))
		require.Empty(t, d.calls)
		in.Grid = -start
		require.NoError(t, s.Step(d, d, in))
		require.Equal(t, Active, s.Phase)
		require.Equal(t, start-100, s.Power())
		d.state.Power = -s.Power()
		in.StartPower = 2000 // editing the UI value does not stop a running charge
		in.Grid = 0
		require.NoError(t, s.Step(d, d, in))
		require.Equal(t, Active, s.Phase)
		require.Equal(t, start-100, s.Power())
	}
}

func TestPVContinuesBelowStartThresholdUntilFullOrImport(t *testing.T) {
	for _, stop := range []string{"full", "import"} {
		t.Run(stop, func(t *testing.T) {
			d := newDevice()
			s := session(d)
			in := inputs()
			require.NoError(t, s.Step(d, d, in))
			d.state.Power = -400
			d.state.Soc = 99.9
			for _, grid := range []float64{-100, -50, -1, 0} {
				in.Grid = grid
				require.NoError(t, s.Step(d, d, in))
				require.Equal(t, Active, s.Phase)
				require.Equal(t, 400.0, s.Power())
			}
			commands := len(d.targets)
			if stop == "full" {
				d.state.Soc = 100
			} else {
				in.Grid = 0.1
			}
			require.NoError(t, s.Step(d, d, in))
			require.Equal(t, Observing, s.Phase)
			require.Equal(t, "release", d.calls[len(d.calls)-1])
			require.Len(t, d.targets, commands, "no further charging command before release")
		})
	}
}

func TestPVBudgetAdjustmentIsNotMeasuredGridImport(t *testing.T) {
	d := newDevice()
	s := session(d)
	in := inputs()
	require.NoError(t, s.Step(d, d, in))
	d.state.Power = -400
	in.Grid = -100
	in.UnavailablePower = 150
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, Active, s.Phase)
	require.Equal(t, 250.0, s.Power())
}
func TestPVLimitsAndPriority(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Input, *fakeDevice)
	}{
		{"full", func(in *Input, d *fakeDevice) { d.state.Soc = 100 }},
		{"configured soc", func(in *Input, d *fakeDevice) { in.MaxSoc = 80; d.state.Soc = 80 }},
		{"invalid meters", func(in *Input, d *fakeDevice) { in.Valid = false }},
		{"tariff or HEMS", func(in *Input, d *fakeDevice) { in.Allowed = false }},
		{"no PV", func(in *Input, d *fakeDevice) { in.PV = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newDevice()
			s := session(d)
			in := inputs()
			require.NoError(t, s.Step(d, d, in))
			tc.change(&in, d)
			require.NoError(t, s.Step(d, d, in))
			require.Equal(t, Observing, s.Phase)
			require.Equal(t, "release", d.calls[len(d.calls)-1])
		})
	}
	d := newDevice()
	d.state.Soc = 95
	s := session(d)
	in := inputs()
	in.Grid = -5000
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, 3300.0, s.Power(), "no arbitrary 80 percent derating")
}
func TestPVLostOwnerCannotOverwriteForeignControl(t *testing.T) {
	d := newDevice()
	s := session(d)
	in := inputs()
	require.NoError(t, s.Step(d, d, in))
	d.state.LeaseID = "another-controller"
	d.state.NativeMode = "foreign"
	d.state.Power = -1200
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, Foreign, s.Phase)
	require.Equal(t, "another-controller", d.state.LeaseID)
	require.Equal(t, "foreign", d.state.NativeMode)
	require.Equal(t, -1200.0, d.state.Power)
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, []string{"acquire", "release"}, d.calls)
}
func TestPVFailedWriteAndReleaseLockout(t *testing.T) {
	d := newDevice()
	s := session(d)
	in := inputs()
	d.writeErr = errors.New("lost ack")
	d.releaseErr = errors.New("offline")
	require.Error(t, s.Step(d, d, in))
	require.Equal(t, Recovering, s.Phase)
	require.Error(t, s.Step(d, d, in))
	require.Equal(t, []string{"acquire", "release", "release"}, d.calls)
	d.releaseErr = nil
	require.NoError(t, s.Step(d, d, in))
	require.Equal(t, Observing, s.Phase)
	require.Empty(t, d.state.LeaseID)
}
func TestPVRestartAndCrashExpiry(t *testing.T) {
	d := newDevice()
	s := session(d)
	in := inputs()
	require.NoError(t, s.Step(d, d, in))
	restarted := session(d)
	require.NoError(t, restarted.Step(d, d, in))
	require.Equal(t, Foreign, restarted.Phase)
	require.Len(t, d.calls, 1)
	d.now = d.now.Add(LeaseDuration + time.Second)
	state, err := d.BatteryControlState()
	require.NoError(t, err)
	require.Equal(t, api.BatteryOperatingAuto, state.Mode)
	require.Equal(t, "10", state.NativeMode)
	require.NoError(t, restarted.Step(d, d, in))
	require.Equal(t, Active, restarted.Phase)
}
func TestPVObservationFailureReleasesOnlyOwnLease(t *testing.T) {
	for _, active := range []bool{false, true} {
		d := newDevice()
		s := session(d)
		in := inputs()
		if active {
			require.NoError(t, s.Step(d, d, in))
		}
		d.readErr = errors.New("offline")
		require.Error(t, s.Step(d, d, in))
		if active {
			require.Equal(t, []string{"acquire", "release"}, d.calls)
		} else {
			require.Empty(t, d.calls)
		}
	}
	d := newDevice()
	s := session(d)
	d.stale = true
	require.NoError(t, s.Step(d, d, inputs()))
	require.Empty(t, d.calls)
}
func TestPVRequiresActualAcknowledgement(t *testing.T) {
	d := newDevice()
	s := session(d)
	d.unacknowledged = true
	require.Error(t, s.Step(d, d, inputs()))
	require.NotEqual(t, Active, s.Phase)
	require.Zero(t, s.Power())
}
func TestPVReadOnlyDeviceCannotBeForced(t *testing.T) {
	d := newDevice()
	s := session(d)
	require.NoError(t, s.Step(d, nil, inputs()))
	require.Equal(t, Blocked, s.Phase)
	require.Empty(t, d.calls)
}

func TestPVIdleOffsetStart(t *testing.T) {
	d := newDevice()
	d.state.Power, d.state.Soc = -3, 0
	s := session(d)
	require.NoError(t, s.Step(d, d, inputs()))
	require.Equal(t, Active, s.Phase)
	require.Equal(t, []float64{400}, d.targets)
}
