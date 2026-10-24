package ecoflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
)

// Update is one piece of telemetry from the wallbox. Exactly one field is set.
type Update struct {
	Heartbeat *Heartbeat // 2/33
	Settings  *Settings  // 241/44, only while the EcoFlow app is active
}

// Settle times after a command, during which telemetry may still show the old
// state. On hardware a start showed after 4-13 s (44 s after a phase switch), a
// stop under 2 s and a setting after 1.5 s in 241/44, with stale reports before.
const (
	startSettle    = 60 * time.Second
	stopSettle     = 30 * time.Second
	settingsSettle = 10 * time.Second
)

// command is the last value written to the wallbox.
type command[T comparable] struct {
	value T
	at    time.Time
	ok    bool
	open  bool // neither confirmed nor contradicted yet
}

// overrides reports whether telemetry received at is late enough to show the
// outcome of the command.
func (c command[T]) overrides(at time.Time, settle time.Duration) bool {
	return !c.ok || at.Sub(c.at) >= settle
}

// write publishes a command to the gateway. It returns once the broker has
// accepted it; the wallbox sends no reply, its effect shows in later telemetry.
func (wb *Wallbox) write(name string, payload func(seq uint64) []byte) error {
	if wb.gateway == "" {
		return errors.New("missing gateway serial")
	}

	seq := wb.conn.nextSeq()
	frame := payload(seq)
	wb.log.DEBUG.Printf("%s: send %s seq=%d", wb.serial, name, seq)
	wb.log.TRACE.Printf("%s: send %x", wb.serial, frame)

	topic := fmt.Sprintf("/app/%s/%s/thing/property/set", wb.conn.userID, wb.gateway)
	if err := wb.conn.publish(topic, frame); err != nil {
		return fmt.Errorf("publish %s: %w", name, err)
	}

	return nil
}

// StartStop starts (241/100 selector 2) or stops (selector 1) charging. It
// does not write when Enabled already reports that state.
func (wb *Wallbox) StartStop(start bool) error {
	if enabled, err := wb.Enabled(); err == nil && enabled == start {
		wb.log.DEBUG.Printf("%s: already enabled=%t, not sending", wb.serial, start)
		return nil
	}

	var selector uint64 = 1
	if start {
		selector = 2
	}

	if err := wb.write(fmt.Sprintf("enable=%t", start), func(seq uint64) []byte {
		return BuildStartStop(wb.serial, selector, seq)
	}); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.release = command[bool]{value: start, at: wb.conn.clock.Now(), ok: true}
	wb.mu.Unlock()

	return nil
}

// Enabled reports whether charging is released. It follows the last start or
// stop until a later heartbeat shows charging or a stop unambiguously; a
// plugged, idle wallbox (state 2) may be either.
func (wb *Wallbox) Enabled() (bool, error) {
	hb, err := wb.heartbeat.Get()
	if err != nil {
		return false, err
	}

	wb.mu.Lock()
	cmd, at := wb.release, wb.seen
	wb.mu.Unlock()

	settle := stopSettle
	if cmd.value {
		settle = startSettle
	}

	if cmd.overrides(at, settle) {
		switch {
		case hb.State.Enabled():
			return true, nil
		case hb.State == StateStoppedByWallbox:
			return false, nil
		}
	}

	return cmd.ok && cmd.value, nil
}

// SetMaxCurrent writes the maximum charge current in 0.1 A (241/102 settings
// field 3).
func (wb *Wallbox) SetMaxCurrent(raw uint64) error {
	if err := wb.write(fmt.Sprintf("maxcurrent=%.1fA", float64(raw)/10), func(seq uint64) []byte {
		return BuildMaxCurrentSet(wb.serial, wb.gateway, raw, seq)
	}); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.current = command[uint64]{value: raw, at: wb.conn.clock.Now(), ok: true}
	wb.mu.Unlock()

	return nil
}

// MaxCurrent returns the maximum charge current in 0.1 A: from 241/44 or
// heartbeat field 18 once they can show the last write, else the written value.
func (wb *Wallbox) MaxCurrent() (uint64, error) {
	hb, err := wb.heartbeat.Get()
	if err != nil {
		return 0, err
	}

	wb.mu.Lock()
	cmd, hbAt := wb.current, wb.seen
	wb.mu.Unlock()

	if s, at, ok := wb.latestSettings(); ok && cmd.overrides(at, settingsSettle) {
		return s.MaxCurrentRaw, nil
	}

	switch {
	case hb.MaxCurrentRaw > 0 && cmd.overrides(hbAt, settingsSettle):
		return hb.MaxCurrentRaw, nil
	case cmd.ok:
		return cmd.value, nil
	default:
		return 0, api.ErrNotAvailable
	}
}

// SetPhaseMode writes PhaseOne or PhaseThree (241/102 settings field 5). It
// does not write when telemetry already shows the mode. Whether the write took
// effect is logged once telemetry shows it.
func (wb *Wallbox) SetPhaseMode(mode uint64) error {
	if mode != PhaseOne && mode != PhaseThree {
		return fmt.Errorf("invalid phase mode: %d", mode)
	}

	if wb.hasPhaseMode(mode) {
		wb.log.DEBUG.Printf("%s: phase mode %d already set, not sending", wb.serial, mode)
		return nil
	}

	if err := wb.write(fmt.Sprintf("phasemode=%d", mode), func(seq uint64) []byte {
		return BuildPhaseSet(wb.serial, wb.gateway, mode, seq)
	}); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.phases = command[uint64]{value: mode, at: wb.conn.clock.Now(), ok: true, open: true}
	wb.mu.Unlock()

	return nil
}

// verifyPhases logs whether telemetry received at confirms or contradicts the
// last phase write. Field 21 = 0 is reported in auto and in 3p alike, so
// without 241/44 a 3p write can only be contradicted, never confirmed.
func (wb *Wallbox) verifyPhases(u Update, at time.Time) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	cmd := wb.phases
	if !cmd.open || !cmd.overrides(at, settingsSettle) {
		return
	}

	var applied bool
	switch {
	case u.Settings != nil:
		applied = u.Settings.PhaseModeRaw == cmd.value
	case u.Heartbeat != nil && u.Heartbeat.HasPhaseMode:
		one := u.Heartbeat.PhaseModeRaw == 1
		if cmd.value == PhaseThree && !one {
			return
		}
		applied = one == (cmd.value == PhaseOne)
	default:
		return
	}

	wb.phases.open = false
	if applied {
		wb.log.DEBUG.Printf("%s: phase mode %d confirmed", wb.serial, cmd.value)
	} else {
		wb.log.WARN.Printf("%s: phase mode %d not applied by the wallbox", wb.serial, cmd.value)
	}
}

// hasPhaseMode reports whether fresh telemetry shows mode selected. Field 21 = 0
// is reported in auto and in 3p alike, so only a 241/44 report shows three phases.
func (wb *Wallbox) hasPhaseMode(mode uint64) bool {
	if s, _, ok := wb.latestSettings(); ok {
		return s.PhaseModeRaw == mode
	}

	hb, ok := wb.latest()
	if !ok || !hb.HasPhaseMode {
		return false
	}

	return mode == PhaseOne && hb.PhaseModeRaw == 1
}
