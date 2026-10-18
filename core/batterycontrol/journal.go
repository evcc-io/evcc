package batterycontrol

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
)

const (
	Conflict             Phase = "control-conflict"
	Disabled             Phase = "disabled"
	Confirming           Phase = "confirming"
	ConfirmationWindow         = 90 * time.Second
	RestoreRetryInterval       = 30 * time.Second
)

// Record stores intent before I/O and confirmation afterwards. It is evidence
// of evcc's actions, not a device-enforced exclusive ownership claim.
type Record struct {
	PendingSince   time.Time `json:"pendingSince,omitempty"`
	RestoreAttempt time.Time `json:"restoreAttempt,omitempty"`
	CommandPending bool      `json:"commandPending,omitempty"`
	UserRecovery   bool      `json:"userRecovery,omitempty"`
	Schema         int       `json:"schema"`
	Revision       string    `json:"revision"`
	Identity       string    `json:"identity"`
	Stage          string    `json:"stage"`
	PreviousMode   string    `json:"previousMode"`
	Power          float64   `json:"power"`
	Updated        time.Time `json:"updated"`
	Reason         string    `json:"reason"`
}

type JournalSession struct {
	journalTrusted bool
	*Session
	Record           Record
	store            JournalStore
	reader           api.BatteryControlStateReader
	setter           api.BatteryChargePowerController
	restorer         api.BatteryControlRestorer
	restart          bool
	storeErr         error
	recoveryOnly     bool
	noChargeSince    time.Time
	ObservationAt    time.Time
	ObservationValid bool
}

func NewJournalSession(session *Session, store JournalStore, reader api.BatteryControlStateReader, setter api.BatteryChargePowerController, restorer api.BatteryControlRestorer) *JournalSession {
	j := &JournalSession{Session: session, store: store, reader: reader, setter: setter, restorer: restorer, restart: true}
	if store == nil {
		j.storeErr = fmt.Errorf("durable journal unavailable")
	} else {
		j.Record, j.storeErr = store.Load()
	}

	if j.storeErr == nil && j.Record.Schema != 0 && (!slices.Contains([]string{"idle", "intent", "active", "releasing", "conflict", "disabled"}, j.Record.Stage) || !finite(j.Record.Power) || j.Record.Power < 0 || j.Record.Identity == "") {
		j.storeErr = fmt.Errorf("invalid journal state")
	}
	j.journalTrusted = j.storeErr == nil
	return j
}

func (j *JournalSession) save(record Record) error {
	if j.storeErr != nil {
		j.transition(Blocked, "journal unavailable")
		return j.storeErr
	}
	record.Schema, record.Revision, record.Updated = 1, rand.Text(), j.clock().UTC()
	if err := j.store.Save(record); err != nil {
		j.storeErr = err
		j.transition(Blocked, "journal write failed")
		return err
	}
	j.Record = record
	return nil
}

func (j *JournalSession) conflict(reason string) error {
	r := j.Record
	r.Stage, r.Reason = "conflict", reason
	err := j.save(r)
	if err == nil {
		j.transition(Conflict, reason)
	}
	return err
}

func (j *JournalSession) clock() time.Time {
	if j.now != nil {
		return j.now()
	}
	return time.Now()
}

func (j *JournalSession) outstanding() bool {
	return j.Record.Stage == "intent" || j.Record.Stage == "active" || j.Record.Stage == "releasing"
}

func (j *JournalSession) observe() (api.BatteryControlState, error) {
	j.ObservationValid = false
	s, err := j.reader.BatteryControlState()
	if err != nil {
		return s, err
	}
	if !fresh(s, j.clock()) || s.Identity == "" || (s.Mode == api.BatteryOperatingAuto && s.NativeMode == "") || (s.ChargeSetpoint != nil && (!finite(*s.ChargeSetpoint) || *s.ChargeSetpoint < 0)) {
		return s, fmt.Errorf("invalid or stale battery observation")
	}
	j.ObservedMode, j.ObservedNativeMode, j.ObservedPower = s.Mode, s.NativeMode, s.Power
	j.ObservedNativeCharging = s.NativeCharging
	j.ObservationAt, j.ObservationValid = s.ObservedAt, true
	return s, nil
}

func (j *JournalSession) finish(reason string) error {
	r := j.Record
	r.Stage, r.Reason, r.Power = "idle", reason, 0
	r.UserRecovery, r.CommandPending = false, false
	r.PendingSince, r.RestoreAttempt = time.Time{}, time.Time{}
	if err := j.save(r); err != nil {
		return err
	}
	j.power = 0
	j.noChargeSince = time.Time{}
	j.transition(Observing, reason)
	return nil
}

// Release also owns unconfirmed commands: a lost or delayed reply does not
// cancel responsibility for restoring the recorded automatic mode.
func (j *JournalSession) Release(reason string) error {
	if !j.journalTrusted {
		return fmt.Errorf("recovery refused: journal could not be validated")
	}
	if !j.outstanding() {
		return nil
	}
	if j.Record.Stage != "releasing" {
		r := j.Record
		r.CommandPending = r.CommandPending || r.Stage == "intent"
		r.Stage, r.Reason = "releasing", reason
		r.RestoreAttempt = time.Time{}
		// A durable intent already exists. Storage failure must not prevent its rollback.
		if err := j.save(r); err != nil {
			j.Record = r
			j.recoveryOnly = true
		}
	}
	j.transition(Recovering, j.Record.Reason)
	state, err := j.observe()
	if err != nil {
		return err
	}
	if state.Identity != j.Record.Identity {
		return j.conflict("device identity changed")
	}
	if state.Mode == api.BatteryOperatingAuto {
		if !j.Record.CommandPending && j.Record.RestoreAttempt.IsZero() {
			return j.finish("automatic operation observed")
		}
		if !j.Record.RestoreAttempt.IsZero() && state.ObservedAt.After(j.Record.RestoreAttempt) {
			if j.Record.PreviousMode != "" && state.NativeMode != j.Record.PreviousMode {
				return j.conflict("different automatic mode after release")
			}
			return j.finish("control returned to automatic operation")
		}
	} else if state.Mode != api.BatteryOperatingManual {
		return fmt.Errorf("recovery pending: operating mode unknown")
	} else if !j.Record.CommandPending && !j.Record.UserRecovery && j.Record.RestoreAttempt.IsZero() && state.ChargeSetpoint != nil && *state.ChargeSetpoint != j.Record.Power {
		return j.conflict("manual setpoint differs from journal")
	}
	if !j.Record.RestoreAttempt.IsZero() && j.clock().Sub(j.Record.RestoreAttempt) >= 0 && j.clock().Sub(j.Record.RestoreAttempt) < RestoreRetryInterval {
		return nil
	}
	r := j.Record
	r.RestoreAttempt = j.clock()
	if err := j.save(r); err != nil {
		j.Record = r
		j.recoveryOnly = true
	}
	j.transition(Recovering, "waiting for automatic mode confirmation")
	if err := j.restorer.RestoreBatteryMode(r.PreviousMode); err != nil {
		return err
	}
	// Confirmation is read on a later cycle, not immediately after the command.
	return nil
}

func (j *JournalSession) confirm(state api.BatteryControlState) error {
	if j.clock().Sub(j.Record.PendingSince) >= ConfirmationWindow {
		return j.Release("charge confirmation timed out")
	}
	if state.Mode != api.BatteryOperatingManual || !state.ObservedAt.After(j.Record.PendingSince) ||
		(state.ChargeSetpoint != nil && *state.ChargeSetpoint != j.Record.Power) {
		j.transition(Confirming, "waiting for manual mode confirmation")
		return nil
	}
	if state.ChargeSetpoint == nil && state.Power >= -NativeChargeIdleTolerance {
		j.transition(Confirming, "manual mode observed; waiting for actual charging")
		return nil
	}
	r := j.Record
	r.Stage, r.Reason, r.CommandPending = "active", "manual mode confirmed after charge command", false
	if err := j.save(r); err != nil {
		return errors.Join(err, j.Release("confirmation journal failed"))
	}
	j.power = r.Power
	j.transition(Active, r.Reason)
	return nil
}

func safePVInput(in Input, state api.BatteryControlState) bool {
	return in.Valid && in.Allowed && finite(in.Grid) && in.Grid <= 0 && finite(in.PV) && in.PV > 0 && finite(in.UnavailablePower) && in.UnavailablePower >= 0 && finite(in.Reserve) && in.Reserve >= 0 && finite(in.StartPower) && in.StartPower > 0 && finite(in.MaxPower) && in.MaxPower > 0 && finite(in.MaxSoc) && in.MaxSoc > 0 && in.MaxSoc <= 100 && state.Soc < in.MaxSoc
}

// Resolve persists the explicit decision, never a device command. A revision
// prevents an old browser dialog from resolving a newer conflict.
func (j *JournalSession) Resolve(action, revision string) error {
	if j.storeErr != nil {
		return j.storeErr
	}
	if revision == "" || revision != j.Record.Revision {
		return fmt.Errorf("stale journal revision; reload battery state")
	}
	if j.Record.Stage != "conflict" && j.Record.Stage != "disabled" {
		return fmt.Errorf("no conflict or disabled control to resolve")
	}
	r := j.Record
	switch action {
	case "disable":
		r.Stage, r.Reason = "disabled", "disabled by user"
	case "recover":
		s, err := j.observe()
		if err != nil {
			return err
		}
		if s.Mode != api.BatteryOperatingAuto && s.Mode != api.BatteryOperatingManual {
			return fmt.Errorf("unknown device mode; recovery refused")
		}
		if r.Identity != s.Identity {
			r.PreviousMode = ""
		}
		r.Identity = s.Identity
		r.Stage, r.Reason = "releasing", "automatic recovery explicitly authorized by user"
		r.UserRecovery = true
		r.RestoreAttempt = time.Time{}
	default:
		return fmt.Errorf("invalid decision")
	}
	if err := j.save(r); err != nil {
		return err
	}
	if action == "disable" {
		j.transition(Disabled, r.Reason)
	} else {
		j.transition(Recovering, r.Reason)
	}
	return nil
}

func (j *JournalSession) Step(in Input) error {
	if j.storeErr != nil && (!j.journalTrusted || !j.outstanding()) {
		j.ObservationValid = false
		j.transition(Blocked, "journal unavailable")
		return j.storeErr
	}
	// Even locked controllers keep observations fresh; the lock still forbids writes.
	if j.Record.Stage == "disabled" || j.Record.Stage == "conflict" {
		_, err := j.observe()
		phase := Conflict
		if j.Record.Stage == "disabled" {
			phase = Disabled
		}
		j.transition(phase, j.Record.Reason)
		return err
	}
	if j.outstanding() && (j.restart || j.storeErr != nil || j.recoveryOnly) {
		j.restart = false
		return j.Release("restart or storage failure: restore previous automatic mode")
	}
	j.restart = false
	if j.Record.Stage == "releasing" {
		return j.Release(j.Record.Reason)
	}
	state, err := j.observe()
	if err != nil {
		if j.outstanding() {
			return errors.Join(err, j.Release("battery observation failed"))
		}
		j.transition(Observing, "battery observation unavailable")
		return err
	}
	s := state
	if j.Record.Identity != "" && j.Record.Identity != s.Identity {
		return j.conflict("device identity changed")
	}
	if j.Record.Identity == "" {
		j.Record.Identity = s.Identity
	}
	// Without setpoint readback, repeated writes must not hide a battery that stopped charging.
	if j.outstanding() && s.Mode == api.BatteryOperatingManual && s.ChargeSetpoint == nil && s.Power >= -NativeChargeIdleTolerance {
		if j.noChargeSince.IsZero() {
			j.noChargeSince = j.clock()
		}
		if j.clock().Sub(j.noChargeSince) >= ConfirmationWindow {
			return j.Release("actual charging absent; restore previous mode")
		}
	} else {
		j.noChargeSince = time.Time{}
	}
	if j.Record.Stage == "intent" {
		if !safePVInput(in, s) || j.Record.Power > in.MaxPower || j.Record.Power > in.PV {
			return j.Release("unsafe budget while command confirmation pending")
		}
		return j.confirm(s)
	}
	owned := j.Record.Stage == "active"
	if owned && s.Mode == api.BatteryOperatingAuto {
		return j.finish("battery returned to automatic operation")
	}
	if owned && s.Mode == api.BatteryOperatingUnknown {
		return j.Release("unknown mode during own control")
	}
	if owned && (s.Mode != api.BatteryOperatingManual || (s.ChargeSetpoint != nil && *s.ChargeSetpoint != j.Record.Power)) {
		return j.conflict("manual state differs from confirmed journal")
	}
	if !owned {
		if s.Mode != api.BatteryOperatingAuto {
			return j.conflict("manual or unknown operation without own journal")
		}
		if nativeCharging(s) {
			j.transition(Native, "battery is charging autonomously")
			return nil
		}
	}
	if !safePVInput(in, s) {
		if owned {
			return j.Release("no safe PV budget, grid import or SoC limit")
		}
		j.transition(Observing, "no safe charging budget")
		return nil
	}
	grid := in.Grid + in.UnavailablePower
	export := -grid - max(0, s.Power)
	if !owned && export < in.StartPower {
		j.transition(Observing, "below configured start threshold")
		return nil
	}
	charge := 0.0
	if owned {
		charge = max(0, -s.Power)
	}
	target := math.Floor(min(in.MaxPower, in.PV, max(0, charge+export-in.Reserve)))
	if owned && grid <= 0 && grid >= -in.Reserve {
		target = min(j.Record.Power, in.MaxPower, in.PV)
	}
	if target <= 0 {
		if owned {
			return j.Release("no usable surplus")
		}
		return nil
	}
	if owned && target == j.Record.Power {
		j.power = target
		j.transition(Active, "journal-confirmed manual control")
		return nil
	}
	r := j.Record
	if !owned {
		r.PreviousMode = s.NativeMode
	}
	r.Stage, r.Power, r.Reason = "intent", target, "charge command prepared"
	r.CommandPending, r.PendingSince, r.RestoreAttempt = true, j.clock(), time.Time{}
	if err := j.save(r); err != nil {
		if owned {
			return errors.Join(err, j.Release("command journal failed"))
		}
		return err
	}
	j.transition(Confirming, "waiting for manual mode confirmation")
	setPower := j.setter.SetBatteryChargePower
	if update, ok := j.setter.(api.BatteryChargePowerUpdater); owned && ok {
		setPower = update.UpdateBatteryChargePower
	}
	if err := setPower(target); err != nil {
		return errors.Join(err, j.Release("charge command failed; restore previous mode"))
	}
	// The status endpoint may lag behind a successful command. Observe next cycle.
	return nil
}

func (j *JournalSession) Close() error {
	if closer, ok := j.store.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// SetDevice refreshes adapters after configuration reload; observation still
// checks the journal identity before any command can reach the replacement.
func (j *JournalSession) SetDevice(reader api.BatteryControlStateReader, setter api.BatteryChargePowerController, restorer api.BatteryControlRestorer) {
	j.reader, j.setter, j.restorer = reader, setter, restorer
}

// ReservedPower includes possibly applied commands while confirmation or rollback is pending.
func (j *JournalSession) ReservedPower() float64 {
	if j.outstanding() {
		return j.Record.Power
	}
	return 0
}

// Shutdown attempts rollback and waits for a later device observation, within the caller's deadline.
func (j *JournalSession) Shutdown(ctx context.Context) error {
	for {
		err := j.Release("evcc shutdown")
		if !j.journalTrusted || !j.outstanding() {
			return err
		}
		select {
		case <-ctx.Done():
			return errors.Join(err, fmt.Errorf("automatic restoration unconfirmed at shutdown: %w", ctx.Err()))
		case <-time.After(time.Second):
		}
	}
}
