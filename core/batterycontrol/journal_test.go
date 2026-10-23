package batterycontrol

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

type memoryJournal struct {
	record    Record
	writes    []Record
	err       error
	failStage string
}

func (m *memoryJournal) Load() (Record, error) { return m.record, m.err }
func (m *memoryJournal) Save(r Record) error {
	if m.err != nil {
		return m.err
	}
	if m.failStage == r.Stage {
		return errors.New("sync failed")
	}
	m.record = r
	m.writes = append(m.writes, r)
	return nil
}

type journalDevice struct {
	state                                                               api.BatteryControlState
	store                                                               *memoryJournal
	commands                                                            []string
	now                                                                 time.Time
	failRead, failWrite, failRestore, delayCharge, delayRestore, frozen bool
}

func (d *journalDevice) BatteryControlState() (api.BatteryControlState, error) {
	if d.failRead {
		return d.state, errors.New("offline")
	}
	s := d.state
	if !d.frozen {
		s.ObservedAt = d.now
	}
	return s, nil
}
func (d *journalDevice) SetBatteryChargePower(p float64) error {
	if d.store.record.Stage != "intent" {
		panic("command before durable intent")
	}
	d.commands = append(d.commands, "charge")
	if !d.delayCharge {
		d.manual(p)
	}
	if d.failWrite {
		return errors.New("lost acknowledgement")
	}
	return nil
}
func (d *journalDevice) manual(p float64) {
	d.state.Mode = api.BatteryOperatingManual
	d.state.NativeMode = "1"
	d.state.ChargeSetpoint = new(p)
}
func (d *journalDevice) automatic(mode string) {
	d.state.Mode = api.BatteryOperatingAuto
	d.state.NativeMode = mode
	d.state.ChargeSetpoint = nil
	d.state.Power = 0
}
func (d *journalDevice) RestoreBatteryMode(mode string) error {
	// A persisted intent remains sufficient evidence when a later journal write fails.
	if d.store.record.Stage != "releasing" && d.store.record.Stage != "intent" && d.store.record.Stage != "active" {
		panic("restore without own durable operation")
	}
	d.commands = append(d.commands, "restore:"+mode)
	if d.failRestore {
		return errors.New("offline")
	}
	if mode == "" {
		mode = "2"
	}
	if !d.delayRestore {
		d.automatic(mode)
	}
	return nil
}
func journalFixture() (*JournalSession, *journalDevice, *memoryJournal) {
	m := new(memoryJournal)
	d := &journalDevice{store: m, now: time.Now(), state: api.BatteryControlState{Identity: "device", Mode: api.BatteryOperatingAuto, NativeMode: "10", Soc: 70}}
	j := NewJournalSession(&Session{now: func() time.Time { return d.now }}, m, d, d, d)
	return j, d, m
}

type updatingJournalDevice struct{ *journalDevice }

func (d updatingJournalDevice) UpdateBatteryChargePower(p float64) error {
	if d.store.record.Stage != "intent" {
		panic("update before durable intent")
	}
	d.commands = append(d.commands, "update")
	d.manual(p)
	if d.failWrite {
		return errors.New("update acknowledgement lost")
	}
	return nil
}

func TestJournalOwnedPowerUpdateDoesNotReenterManualMode(t *testing.T) {
	for _, fail := range []bool{false, true} {
		j, d, _ := journalFixture()
		j.setter = updatingJournalDevice{d}
		startJournal(t, j, d)
		d.state.Power = -410
		d.failWrite = fail
		in := inputs()
		in.Grid = -118
		err := tick(j, d, in)
		if fail {
			require.Error(t, err)
			require.Equal(t, []string{"charge", "update", "restore:10"}, d.commands)
			require.Equal(t, Recovering, j.Phase)
		} else {
			require.NoError(t, err)
			require.Equal(t, []string{"charge", "update"}, d.commands)
			require.Equal(t, 428.0, j.Record.Power)
			require.Equal(t, Confirming, j.Phase)
			require.NoError(t, tick(j, d, in))
			require.Equal(t, Active, j.Phase)
		}
	}
}

func TestJournalManualModeAloneDoesNotConfirmCharging(t *testing.T) {
	for _, resume := range []bool{false, true} {
		j, d, _ := journalFixture()
		require.NoError(t, j.Step(inputs()))
		d.state.ChargeSetpoint = nil
		d.state.Power = -3
		require.NoError(t, tick(j, d, inputs()))
		require.Equal(t, Confirming, j.Phase)
		require.Equal(t, []string{"charge"}, d.commands)
		if resume {
			d.state.Power = -399
			require.NoError(t, tick(j, d, inputs()))
			require.Equal(t, Active, j.Phase)
		} else {
			require.NoError(t, tick(j, d, inputs()))
			require.Equal(t, Confirming, j.Phase)
			require.NoError(t, tick(j, d, inputs()))
			require.Equal(t, Recovering, j.Phase)
			require.Equal(t, []string{"charge", "restore:10"}, d.commands)
		}
	}
}

func TestJournalSetpointUpdaterRequiresOwnedControl(t *testing.T) {
	for _, restart := range []bool{false, true} {
		j, d, m := journalFixture()
		if restart {
			startJournal(t, j, d)
			j = restartedJournal(j, d, m)
			d.commands = nil
		} else {
			d.manual(400)
		}
		j.setter = updatingJournalDevice{d}
		require.NoError(t, tick(j, d, inputs()))
		if restart {
			require.Equal(t, []string{"restore:10"}, d.commands)
		} else {
			require.Empty(t, d.commands)
			require.Equal(t, Conflict, j.Phase)
		}
	}
}

func TestJournalMissingChargeDoesNotDelayGridImportRecovery(t *testing.T) {
	j, d, _ := journalFixture()
	require.NoError(t, j.Step(inputs()))
	d.state.ChargeSetpoint = nil
	d.state.Power = -3
	in := inputs()
	in.Grid = 10
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Recovering, j.Phase)
	require.Equal(t, []string{"charge", "restore:10"}, d.commands)
}

func TestJournalMissingChargingDuringActiveControlRestores(t *testing.T) {
	j, d, _ := journalFixture()
	startJournal(t, j, d)
	d.state.ChargeSetpoint = nil
	d.state.Power = -3
	in := inputs()
	in.Grid = -50 // deadband holds the previous request, but must not mask missing charge
	for range 3 {
		require.NoError(t, tick(j, d, in))
		require.Equal(t, Active, j.Phase)
	}
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Recovering, j.Phase)
	require.Equal(t, []string{"charge", "restore:10"}, d.commands)
}
func tick(j *JournalSession, d *journalDevice, in Input) error {
	d.now = d.now.Add(30 * time.Second)
	return j.Step(in)
}
func startJournal(t *testing.T, j *JournalSession, d *journalDevice) {
	t.Helper()
	require.NoError(t, j.Step(inputs()))
	require.Equal(t, Confirming, j.Phase)
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, Active, j.Phase)
}
func restartedJournal(j *JournalSession, d *journalDevice, m *memoryJournal) *JournalSession {
	return NewJournalSession(&Session{now: j.now}, m, d, d, d)
}

func TestJournalDelayedChargeAndReturn(t *testing.T) {
	j, d, m := journalFixture()
	d.delayCharge = true
	require.NoError(t, j.Step(inputs()))
	require.Equal(t, "intent", m.record.Stage)
	require.Equal(t, "10", m.record.PreviousMode)
	require.Equal(t, Confirming, j.Phase)
	require.NoError(t, tick(j, d, inputs())) // still automatic
	require.Equal(t, Confirming, j.Phase)
	require.Equal(t, []string{"charge"}, d.commands)
	d.manual(400)
	d.state.Power = -3
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, Active, j.Phase)
	require.Equal(t, 400.0, j.Power())
	d.delayRestore = true
	in := inputs()
	in.Grid = 9000
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Recovering, j.Phase)
	require.Equal(t, "releasing", m.record.Stage)
	// An immediate repeat neither treats manual status as a conflict nor resends.
	require.NoError(t, j.Step(in))
	require.Equal(t, []string{"charge", "restore:10"}, d.commands)
	d.automatic("10")
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Observing, j.Phase)
	require.Equal(t, "idle", m.record.Stage)
}
func TestJournalPendingSafetyAndTimeout(t *testing.T) {
	for _, why := range []string{"timeout", "grid", "permission", "invalid", "full", "limit"} {
		t.Run(why, func(t *testing.T) {
			j, d, _ := journalFixture()
			d.delayCharge = true
			require.NoError(t, j.Step(inputs()))
			in := inputs()
			switch why {
			case "timeout":
				d.now = d.now.Add(ConfirmationWindow)
			case "grid":
				in.Grid = 1
			case "permission":
				in.Allowed = false
			case "invalid":
				in.Valid = false
			case "full":
				d.state.Soc = 100
			case "limit":
				in.MaxPower = 300
			}
			require.NoError(t, j.Step(in))
			require.Equal(t, Recovering, j.Phase)
			require.Equal(t, []string{"charge", "restore:10"}, d.commands)
			require.NoError(t, tick(j, d, in))
			require.Equal(t, "idle", j.Record.Stage)
		})
	}
}
func TestJournalRestartRestoresEveryOwnStage(t *testing.T) {
	for _, stage := range []string{"intent", "active", "releasing"} {
		t.Run(stage, func(t *testing.T) {
			j, d, m := journalFixture()
			require.NoError(t, j.Step(inputs()))
			if stage != "intent" {
				require.NoError(t, tick(j, d, inputs()))
			}
			if stage == "releasing" {
				d.delayRestore = true
				require.NoError(t, j.Release("stop"))
			}
			j = restartedJournal(j, d, m)
			d.now = d.now.Add(31 * time.Second)
			require.NoError(t, j.Step(inputs()))
			require.Equal(t, Recovering, j.Phase)
			d.automatic("10")
			require.NoError(t, tick(j, d, inputs()))
			require.Equal(t, "idle", m.record.Stage)
			require.Equal(t, "charge", d.commands[0])
			for _, c := range d.commands[1:] {
				require.Equal(t, "restore:10", c)
			}
		})
	}
}
func TestJournalWriteFailureReadFailureAndRetry(t *testing.T) {
	for _, failure := range []string{"write", "read", "restore"} {
		t.Run(failure, func(t *testing.T) {
			j, d, m := journalFixture()
			if failure == "write" {
				d.failWrite = true
				require.Error(t, j.Step(inputs()))
			} else {
				startJournal(t, j, d)
				if failure == "read" {
					d.failRead = true
					require.Error(t, tick(j, d, inputs()))
				} else {
					d.failRestore = true
					require.Error(t, j.Release("test"))
				}
			}
			require.Equal(t, "releasing", m.record.Stage)
			require.NotEqual(t, Conflict, j.Phase)
			d.failRead, d.failWrite, d.failRestore = false, false, false
			for range 3 {
				require.NoError(t, tick(j, d, inputs()))
				if m.record.Stage == "idle" {
					break
				}
			}
			require.Equal(t, "idle", m.record.Stage)
		})
	}
}
func TestJournalStorageFailureStillRestoresOwnCommand(t *testing.T) {
	j, d, m := journalFixture()
	m.err = errors.New("disk full")
	require.Error(t, j.Step(inputs()))
	require.Empty(t, d.commands)
	j, d, m = journalFixture()
	require.NoError(t, j.Step(inputs()))
	m.failStage = "active"
	require.Error(t, tick(j, d, inputs()))
	require.Equal(t, []string{"charge", "restore:10"}, d.commands)
	require.Equal(t, "intent", m.record.Stage, "last durable evidence survives failed confirmation")
	m.failStage = ""
	j = restartedJournal(j, d, m)
	require.NoError(t, tick(j, d, inputs()))
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, "idle", m.record.Stage)
}
func TestJournalUnknownManualAndRealConflictStayLockedButObserve(t *testing.T) {
	j, d, m := journalFixture()
	d.manual(418)
	require.NoError(t, j.Step(inputs()))
	require.Equal(t, Conflict, j.Phase)
	d.state.Power = -419
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, -419.0, j.ObservedPower)
	require.True(t, j.ObservationValid)
	require.Empty(t, d.commands)
	d.failRead = true
	require.Error(t, tick(j, d, inputs()))
	require.False(t, j.ObservationValid)
	d.failRead = false
	require.Error(t, j.Resolve("recover", "stale"))
	require.NoError(t, j.Resolve("disable", m.record.Revision))
	j = restartedJournal(j, d, m)
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, Disabled, j.Phase)
	require.NoError(t, j.Resolve("recover", m.record.Revision))
	require.Empty(t, d.commands)
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, Recovering, j.Phase)
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, Observing, j.Phase)
	require.Equal(t, []string{"restore:"}, d.commands)
}
func TestJournalForeignSetpointAndIdentityAreNotRollbackPermission(t *testing.T) {
	for _, change := range []string{"identity", "setpoint"} {
		t.Run(change, func(t *testing.T) {
			j, d, _ := journalFixture()
			startJournal(t, j, d)
			if change == "identity" {
				d.state.Identity = "other"
			} else {
				d.state.ChargeSetpoint = new(999.0)
			}
			require.NoError(t, tick(j, d, inputs()))
			require.Equal(t, Conflict, j.Phase)
			require.NoError(t, j.Release("shutdown"))
			require.Equal(t, []string{"charge"}, d.commands)
		})
	}
}
func TestJournalObservedAutomaticReturnAndNativeCharging(t *testing.T) {
	j, d, _ := journalFixture()
	d.state.NativeCharging = true
	d.state.Power = -900
	for range 4 {
		require.NoError(t, tick(j, d, inputs()))
	}
	require.Equal(t, Native, j.Phase)
	require.Empty(t, d.commands)
	d.state.NativeCharging = false
	d.state.Power = 0
	startJournal(t, j, d)
	d.automatic("2")
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, Observing, j.Phase)
	require.Equal(t, []string{"charge"}, d.commands)
}
func TestJournalDynamicBudgetAndMeasuredPower(t *testing.T) {
	j, d, _ := journalFixture()
	startJournal(t, j, d)
	d.state.Power = -420
	in := inputs()
	in.Grid = -1500
	in.PV = 3000
	require.NoError(t, tick(j, d, in))
	require.Equal(t, 1820.0, j.Record.Power)
	require.Equal(t, Confirming, j.Phase)
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Active, j.Phase)
	// Actual watts can lag or taper without implying a foreign setpoint.
	d.state.Power = -1500
	in.Grid = 0
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Active, j.Phase)
	require.Equal(t, 1820.0, j.Power())
	in.Grid = 9000
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Recovering, j.Phase)
	require.NoError(t, tick(j, d, in))
	require.Equal(t, "idle", j.Record.Stage)
}
func TestJournalIdleOffsetAndStartSetting(t *testing.T) {
	for _, tc := range []struct {
		power, export, start float64
		flag                 bool
		want                 Phase
	}{
		{-3, 500, 500, false, Confirming}, {-3, 499, 500, false, Observing}, {-5, 500, 500, false, Confirming},
		{-6, 500, 500, false, Native}, {-3, 500, 500, true, Native}, {-900, 500, 500, false, Native},
		{-3, 749, 750, false, Observing}, {-3, 750, 750, false, Confirming},
	} {
		j, d, _ := journalFixture()
		d.state.Power, d.state.Soc, d.state.NativeCharging = tc.power, 0, tc.flag
		in := inputs()
		in.Grid, in.StartPower = -tc.export, tc.start
		require.NoError(t, j.Step(in))
		require.Equal(t, tc.want, j.Phase)
		if tc.want == Confirming {
			require.Equal(t, tc.export-100, j.Record.Power)
		} else {
			require.Empty(t, d.commands)
		}
	}
}
func TestJournalFreshConfirmationRequired(t *testing.T) {
	j, d, _ := journalFixture()
	require.NoError(t, j.Step(inputs()))
	d.frozen = true
	d.state.ObservedAt = d.now
	require.NoError(t, j.Step(inputs()))
	require.Equal(t, Confirming, j.Phase)
	d.frozen = false
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, Active, j.Phase)
}
func TestFileJournalRoundTripAndTornTail(t *testing.T) {
	f := FileJournal{Path: filepath.Join(t.TempDir(), "journal.jsonl")}
	r, err := f.Load()
	require.NoError(t, err)
	require.Empty(t, r.Stage)
	want := Record{Schema: 1, Revision: "one", Stage: "intent", Identity: "device", Power: 400}
	require.NoError(t, f.Save(want))
	want.Revision, want.Stage = "two", "active"
	require.NoError(t, f.Save(want))
	r, err = f.Load()
	require.NoError(t, err)
	require.Equal(t, want, r)
	file, err := os.OpenFile(f.Path, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = file.WriteString(`{"schema":1`)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	_, err = f.Load()
	require.Error(t, err, "never guess through a torn journal tail")
}

func TestFileJournalExclusiveLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	first, err := OpenFileJournal(path)
	require.NoError(t, err)
	_, err = OpenFileJournal(path)
	require.Error(t, err)
	require.NoError(t, first.Close())
	second, err := OpenFileJournal(path)
	require.NoError(t, err)
	require.NoError(t, second.Close())
}

func TestJournalCorruptLoadCannotAuthorizeRollback(t *testing.T) {
	j, d, m := journalFixture()
	startJournal(t, j, d)
	m.err = errors.New("torn final record")
	j = restartedJournal(j, d, m)
	require.Error(t, j.Step(inputs()))
	require.Equal(t, Blocked, j.Phase)
	require.Error(t, j.Release("shutdown"))
	require.Equal(t, []string{"charge"}, d.commands)
}
func TestJournalShutdownFromUnconfirmedCommand(t *testing.T) {
	j, d, m := journalFixture()
	require.NoError(t, j.Step(inputs()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, j.Shutdown(ctx))
	require.Equal(t, []string{"charge", "restore:10"}, d.commands)
	require.Equal(t, "releasing", m.record.Stage, "unconfirmed restoration remains durable")
	j = restartedJournal(j, d, m)
	require.NoError(t, tick(j, d, inputs()))
	require.Equal(t, "idle", m.record.Stage)
}
func TestJournalReleaseRetriesWithSpacing(t *testing.T) {
	j, d, _ := journalFixture()
	startJournal(t, j, d)
	d.delayRestore = true
	require.NoError(t, j.Release("grid import"))
	for range 4 {
		d.now = d.now.Add(time.Second)
		require.NoError(t, j.Step(inputs()))
	}
	require.Equal(t, []string{"charge", "restore:10"}, d.commands)
	d.now = d.now.Add(RestoreRetryInterval)
	require.NoError(t, j.Step(inputs()))
	require.Equal(t, []string{"charge", "restore:10", "restore:10"}, d.commands)
	require.Equal(t, Recovering, j.Phase)
}
func TestJournalDelayedSetpointUpdateAndObservationFailure(t *testing.T) {
	j, d, _ := journalFixture()
	startJournal(t, j, d)
	d.delayCharge = true
	d.state.Power = -400
	in := inputs()
	in.Grid = -1500
	require.NoError(t, tick(j, d, in))
	require.Equal(t, 1800.0, j.Record.Power)
	require.NoError(t, tick(j, d, in))
	require.Equal(t, Confirming, j.Phase, "old setpoint is not foreign control during own update")
	d.failRead = true
	require.Error(t, tick(j, d, in))
	require.Equal(t, "releasing", j.Record.Stage)
	d.failRead = false
	require.NoError(t, tick(j, d, in))
	require.Equal(t, "restore:10", d.commands[len(d.commands)-1])
	require.NoError(t, tick(j, d, in))
	require.Equal(t, "idle", j.Record.Stage)
}

type advancingJournalReader struct{ *journalDevice }

func (r advancingJournalReader) BatteryControlState() (api.BatteryControlState, error) {
	r.now = r.now.Add(time.Second)
	return r.journalDevice.BatteryControlState()
}
func TestJournalShutdownWaitsForLaterObservation(t *testing.T) {
	j, d, m := journalFixture()
	require.NoError(t, j.Step(inputs()))
	j.reader = advancingJournalReader{d}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, j.Shutdown(ctx))
	require.Equal(t, "idle", m.record.Stage)
	require.Equal(t, []string{"charge", "restore:10"}, d.commands)
}
func TestJournalUnconfirmedCommandReservesBudget(t *testing.T) {
	j, d, _ := journalFixture()
	d.delayCharge = true
	require.NoError(t, j.Step(inputs()))
	require.Zero(t, j.Power(), "controller is not active yet")
	require.Equal(t, 400.0, j.ReservedPower(), "possibly applied command must not be allocated twice")
	require.NoError(t, j.Release("abort"))
	require.Equal(t, 400.0, j.ReservedPower())
	require.NoError(t, tick(j, d, inputs()))
	require.Zero(t, j.ReservedPower())
}
