package ecoflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
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

// phaseSettle is the wait between the phase write of a switch while charging
// and the start. On hardware 241/44 showed 1p after 1.1 s and 3p after 5 s.
const phaseSettle = 5 * time.Second

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

// reading returns the written current when v is within 0.1 A of it, else v.
// Heartbeat field 17 showed 159 after a write of 160.
func reading(cmd command[uint64], v uint64) uint64 {
	if cmd.ok && (v == cmd.value || v+1 == cmd.value || v == cmd.value+1) {
		return cmd.value
	}
	return v
}

// write publishes a command to the device with the serial target, the gateway
// or the wallbox itself. It returns once the broker has accepted it; the
// wallbox sends no reply, its effect shows in later telemetry.
func (wb *Wallbox) write(name, target string, payload func(seq uint64) []byte) error {
	if target == "" {
		return errors.New("missing gateway serial")
	}

	seq := wb.conn.nextSeq()
	frame := payload(seq)
	wb.log.DEBUG.Printf("%s: send %s seq=%d", wb.serial, name, seq)
	wb.log.TRACE.Printf("%s: send %x", wb.serial, frame)

	topic := fmt.Sprintf("/app/%s/%s/thing/property/set", wb.conn.userID, target)
	if err := wb.conn.publish(topic, frame); err != nil {
		return fmt.Errorf("publish %s: %w", name, err)
	}

	return nil
}

// StartStop starts (241/100 selector 2) or stops (selector 1) charging. It
// does not write when Enabled already reports that state. During a phase
// switch it only decides whether the switch ends with a start.
func (wb *Wallbox) StartStop(start bool) error {
	wb.cmd.Lock()
	defer wb.cmd.Unlock()

	wb.mu.Lock()
	sw := wb.switching
	if sw != nil {
		sw.restart = start
	}
	wb.mu.Unlock()

	if sw != nil {
		wb.log.DEBUG.Printf("%s: phase switch under way, enable=%t after it", wb.serial, start)
		return nil
	}

	if enabled, err := wb.Enabled(); err == nil && enabled == start {
		wb.log.DEBUG.Printf("%s: already enabled=%t, not sending", wb.serial, start)
		return nil
	}

	return wb.startStop(start)
}

// startStop writes a start or stop. The caller holds wb.cmd.
func (wb *Wallbox) startStop(start bool) error {
	var selector uint64 = 1
	if start {
		selector = 2
	}

	if err := wb.write(fmt.Sprintf("enable=%t", start), wb.gateway, func(seq uint64) []byte {
		return BuildStartStop(wb.serial, selector, seq)
	}); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.release = command[bool]{value: start, at: wb.conn.clock.Now(), ok: true, open: true}
	wb.mu.Unlock()

	return nil
}

// Enabled reports whether charging is released. It follows the last start or
// stop until a later heartbeat shows charging or a stop unambiguously; a
// plugged, idle wallbox (state 2) may be either. During a phase switch it
// reports whether the switch ends with a start.
func (wb *Wallbox) Enabled() (bool, error) {
	hb, err := wb.heartbeat.Get()
	if err != nil {
		return false, err
	}

	wb.mu.Lock()
	cmd, at := wb.release, wb.seen
	sw := wb.switching
	var restart bool
	if sw != nil {
		restart = sw.restart
	}
	wb.mu.Unlock()

	if sw != nil {
		return restart, nil
	}

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

// verifyRelease logs whether a heartbeat received at confirms or contradicts
// the last start or stop. A plugged, idle wallbox (state 2) shows neither.
func (wb *Wallbox) verifyRelease(hb Heartbeat, at time.Time) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	cmd := wb.release
	settle := stopSettle
	if cmd.value {
		settle = startSettle
	}
	if !cmd.open || !cmd.overrides(at, settle) {
		return
	}

	var shown bool
	switch {
	case hb.State.Enabled():
		shown = true
	case hb.State == StateStoppedByWallbox:
		shown = false
	default:
		return
	}

	wb.release.open = false
	if shown == cmd.value {
		wb.log.DEBUG.Printf("%s: enable=%t confirmed", wb.serial, cmd.value)
	} else {
		wb.log.WARN.Printf("%s: enable=%t not applied by the wallbox, state %d", wb.serial, cmd.value, hb.State)
	}
}

// SetCurrent writes the session current setpoint in 0.1 A (ChargerCtrl 2/81 to
// the topic of the wallbox, shown in heartbeat field 17 and 241/44 UserCurrentRaw).
// The configured maximum (field 18) is not written: lowering it while charging
// faulted the wallbox on hardware (state 9, fault 603, only a power cycle
// clears it), whereas the setpoint follows within a minute. The write switches
// the work mode to custom. Written while unplugged, it is stored and applies
// on the next plug-in, where the wallbox starts charging by itself. The
// wallbox does not store a setpoint above its maximum, so that is refused.
func (wb *Wallbox) SetCurrent(raw uint64) error {
	if max := wb.Maximum(); max > 0 && raw > max {
		return fmt.Errorf("current %.1f A above the wallbox maximum %.1f A", float64(raw)/10, float64(max)/10)
	}

	if err := wb.write(fmt.Sprintf("current=%.1fA", float64(raw)/10), wb.serial, func(seq uint64) []byte {
		return BuildChargeCurrent(wb.serial, raw, seq)
	}); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.current = command[uint64]{value: raw, at: wb.conn.clock.Now(), ok: true, open: true}
	wb.mu.Unlock()

	return nil
}

// Maximum returns the configured maximum current in 0.1 A from fresh 241/44 or
// heartbeat field 18, or 0 when unknown.
func (wb *Wallbox) Maximum() uint64 {
	if s, _, ok := wb.latestSettings(); ok {
		return s.MaxCurrentRaw
	}
	if hb, ok := wb.latest(); ok {
		return hb.MaxCurrentRaw
	}
	return 0
}

// Current returns the session current setpoint in 0.1 A: from 241/44 (custom
// mode) or heartbeat field 17 once they can show the last write, else the
// written value. Telemetry within 0.1 A of the write counts as the write.
func (wb *Wallbox) Current() (uint64, error) {
	hb, err := wb.heartbeat.Get()
	if err != nil {
		return 0, err
	}

	wb.mu.Lock()
	cmd, hbAt := wb.current, wb.seen
	wb.mu.Unlock()

	if s, at, ok := wb.latestSettings(); ok && s.WorkMode == WorkModeCustom && cmd.overrides(at, settingsSettle) {
		return reading(cmd, s.UserCurrentRaw), nil
	}

	switch {
	case hb.ChargeCurrentRaw > 0 && cmd.overrides(hbAt, settingsSettle):
		return reading(cmd, hb.ChargeCurrentRaw), nil
	case cmd.ok:
		return cmd.value, nil
	default:
		return 0, api.ErrNotAvailable
	}
}

// verifyCurrent logs whether telemetry received at confirms or contradicts the
// last current write. The wallbox sends no reply to it.
func (wb *Wallbox) verifyCurrent(u Update, at time.Time) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	cmd := wb.current
	if !cmd.open || !cmd.overrides(at, settingsSettle) {
		return
	}

	var shown uint64
	switch {
	case u.Settings != nil && u.Settings.WorkMode == WorkModeCustom:
		shown = u.Settings.UserCurrentRaw
	case u.Heartbeat != nil && u.Heartbeat.ChargeCurrentRaw > 0:
		shown = u.Heartbeat.ChargeCurrentRaw
	default:
		return
	}

	wb.current.open = false
	if reading(cmd, shown) == cmd.value {
		wb.log.DEBUG.Printf("%s: current %.1f A confirmed", wb.serial, float64(cmd.value)/10)
	} else {
		wb.log.WARN.Printf("%s: current %.1f A not applied by the wallbox, shows %.1f A", wb.serial, float64(cmd.value)/10, float64(shown)/10)
	}
}

// SetPhaseMode writes PhaseOne or PhaseThree (241/102 settings field 5). It
// does not write when telemetry already shows the mode. Whether the write took
// effect is logged once telemetry shows it. A charging wallbox is stopped
// first, see switchWhileCharging.
func (wb *Wallbox) SetPhaseMode(mode uint64) error {
	if mode != PhaseOne && mode != PhaseThree {
		return fmt.Errorf("invalid phase mode: %d", mode)
	}

	hb, err := wb.heartbeat.Get()
	if err != nil {
		return err
	}

	wb.cmd.Lock()
	defer wb.cmd.Unlock()

	wb.mu.Lock()
	sw := wb.switching
	pending := sw != nil && !sw.written
	if pending {
		sw.mode = mode
	}
	wb.mu.Unlock()

	if pending {
		return nil
	}

	if wb.hasPhaseMode(mode) {
		wb.log.DEBUG.Printf("%s: phase mode %d already set, not sending", wb.serial, mode)
		return nil
	}

	if hb.State == StateCharging {
		return wb.switchWhileCharging(mode)
	}

	return wb.writePhaseMode(mode)
}

// writePhaseMode writes the phase mode. The caller holds wb.cmd.
func (wb *Wallbox) writePhaseMode(mode uint64) error {
	if err := wb.write(fmt.Sprintf("phasemode=%d", mode), wb.gateway, func(seq uint64) []byte {
		return BuildPhaseSet(wb.serial, wb.gateway, mode, seq)
	}); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.phases = command[uint64]{value: mode, at: wb.conn.clock.Now(), ok: true, open: true}
	wb.mu.Unlock()

	return nil
}

// phaseSwitch is a phase switch while charging.
type phaseSwitch struct {
	mode    uint64
	restart bool          // start once the phase mode is written
	written bool          // phase mode written, start pending
	stop    chan struct{} // telemetry shows the stop
}

// switchWhileCharging stops the wallbox and returns. The phase mode is written
// in the background once a heartbeat shows the stop, the start follows after
// phaseSettle when charging was released. The wallbox was only seen switching
// after a stop, and a start ends the stop. The caller holds wb.cmd.
func (wb *Wallbox) switchWhileCharging(mode uint64) error {
	restart, err := wb.Enabled()
	if err != nil {
		return err
	}

	sw := &phaseSwitch{mode: mode, restart: restart, stop: make(chan struct{}, 1)}
	timeout := wb.conn.clock.Timer(stopSettle)

	wb.mu.Lock()
	wb.switching = sw
	wb.mu.Unlock()

	// a released wallbox needs the stop, a stopped one still reporting charging has it
	if restart {
		if err := wb.startStop(false); err != nil {
			timeout.Stop()
			wb.mu.Lock()
			wb.switching = nil
			wb.mu.Unlock()
			return err
		}
	}

	wb.log.DEBUG.Printf("%s: charging, phase mode %d after the stop", wb.serial, mode)
	go wb.finishSwitch(sw, timeout)

	return nil
}

// stopped tells a phase switch that telemetry shows the wallbox not charging.
func (wb *Wallbox) stopped(state State) {
	if state == StateCharging {
		return
	}

	wb.mu.Lock()
	sw := wb.switching
	wb.mu.Unlock()

	if sw != nil {
		select {
		case sw.stop <- struct{}{}:
		default:
		}
	}
}

// finishSwitch writes the phase mode of sw once the wallbox reports the stop
// and starts it again if wanted. Without a reported stop within the timeout it
// does not write the phase mode under load.
func (wb *Wallbox) finishSwitch(sw *phaseSwitch, timeout *clock.Timer) {
	defer timeout.Stop()
	done := wb.conn.ctx.Done()

	select {
	case <-sw.stop:
		wb.cmd.Lock()
		wb.mu.Lock()
		mode := sw.mode
		sw.written = true
		wb.mu.Unlock()

		settle := wb.conn.clock.Timer(phaseSettle)
		defer settle.Stop()

		if !wb.hasPhaseMode(mode) {
			if err := wb.writePhaseMode(mode); err != nil {
				wb.log.ERROR.Printf("%s: phase switch: %v", wb.serial, err)
			}
		}
		wb.cmd.Unlock()

		select {
		case <-settle.C:
		case <-done:
			return
		}

	case <-timeout.C:
		wb.log.WARN.Printf("%s: stop not reported within %v, phase mode not written", wb.serial, stopSettle)

	case <-done:
		return
	}

	wb.cmd.Lock()
	defer wb.cmd.Unlock()

	wb.mu.Lock()
	restart := sw.restart
	wb.mu.Unlock()

	if restart {
		if err := wb.startStop(true); err != nil {
			wb.log.ERROR.Printf("%s: phase switch: %v", wb.serial, err)
		}
	}

	wb.mu.Lock()
	wb.switching = nil
	wb.mu.Unlock()
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
