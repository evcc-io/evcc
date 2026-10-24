package ecoflow

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

// Update is one piece of telemetry from the wallbox. Exactly one field is set.
type Update struct {
	Heartbeat *Heartbeat // 2/33
	Settings  *Settings  // 241/44, only while the EcoFlow app is active
}

// Confirmation reports whether an update shows that a write took effect.
type Confirmation func(Update) bool

// waiters are the writes waiting for their confirmation.
type waiters struct {
	mu   sync.Mutex
	list []*waiter
}

type waiter struct {
	confirmed Confirmation
	done      chan struct{}
}

// add registers a waiter that is released by the first matching update.
func (ws *waiters) add(confirmed Confirmation) *waiter {
	w := &waiter{confirmed: confirmed, done: make(chan struct{})}
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.list = append(ws.list, w)
	return w
}

// remove unregisters w.
func (ws *waiters) remove(w *waiter) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for i, x := range ws.list {
		if x == w {
			ws.list = append(ws.list[:i], ws.list[i+1:]...)
			return
		}
	}
}

// notify releases and removes every waiter the update confirms.
func (ws *waiters) notify(u Update) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	keep := ws.list[:0]
	for _, w := range ws.list {
		if w.confirmed(u) {
			close(w.done)
		} else {
			keep = append(keep, w)
		}
	}
	ws.list = keep
}

// write publishes a command to the gateway and, unless confirmed is nil, waits
// for telemetry after the publish that satisfies it; else api.ErrTimeout.
func (wb *Wallbox) write(name string, payload func(seq uint64) []byte, confirmed Confirmation, timeout time.Duration) error {
	if wb.gateway == "" {
		return errors.New("missing gateway serial")
	}

	// register before publishing, the confirmation may arrive before Publish returns
	var w *waiter
	if confirmed != nil {
		w = wb.waiters.add(confirmed)
		defer wb.waiters.remove(w)
	}

	seq := wb.conn.nextSeq()
	frame := payload(seq)
	wb.log.DEBUG.Printf("%s: send %s seq=%d", wb.serial, name, seq)
	wb.log.TRACE.Printf("%s: send %x", wb.serial, frame)

	topic := fmt.Sprintf("/app/%s/%s/thing/property/set", wb.conn.userID, wb.gateway)
	if err := wb.conn.transport.Publish(topic, 1, frame); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	if w == nil {
		wb.log.DEBUG.Printf("%s: %s sent, not awaiting confirmation", wb.serial, name)
		return nil
	}

	start := time.Now()
	select {
	case <-w.done:
		wb.log.DEBUG.Printf("%s: %s confirmed after %v", wb.serial, name, time.Since(start).Round(time.Millisecond))
		return nil
	case <-time.After(timeout):
		wb.log.WARN.Printf("%s: %s not confirmed within %v", wb.serial, name, timeout)
		return api.ErrTimeout
	}
}

// StartStop starts (241/100 selector 2) or stops (selector 1) charging and waits
// for a heartbeat showing it. It does not write when already in that state.
func (wb *Wallbox) StartStop(start bool, timeout time.Duration) error {
	hb, ok := wb.latest()
	if ok && hb.State.Enabled() == start {
		wb.log.DEBUG.Printf("%s: already enabled=%t (state %d), not sending", wb.serial, start, hb.State)
		return nil
	}

	var selector uint64 = 1
	if start {
		selector = 2
	}
	payload := func(seq uint64) []byte {
		return BuildStartStop(wb.serial, selector, seq)
	}

	// without a vehicle the start cannot show, so it is not awaited
	name := fmt.Sprintf("enable=%t", start)
	if ok && hb.State == StateUnplugged {
		return wb.write(name, payload, nil, 0)
	}

	return wb.write(name, payload, func(u Update) bool {
		return u.Heartbeat != nil && u.Heartbeat.State.Enabled() == start
	}, timeout)
}

// SetMaxCurrent writes the maximum charge current in 0.1 A (241/102 settings
// field 3) and waits for 241/44 or heartbeat field 18 to show it.
func (wb *Wallbox) SetMaxCurrent(raw uint64, timeout time.Duration) error {
	return wb.write(fmt.Sprintf("maxcurrent=%.1fA", float64(raw)/10), func(seq uint64) []byte {
		return BuildMaxCurrentSet(wb.serial, wb.gateway, raw, seq)
	}, func(u Update) bool {
		switch {
		case u.Settings != nil:
			return u.Settings.MaxCurrentRaw == raw
		case u.Heartbeat != nil:
			return u.Heartbeat.MaxCurrentRaw == raw
		default:
			return false
		}
	}, timeout)
}

// SetPhaseMode writes PhaseOne or PhaseThree (241/102 settings field 5) and waits
// for 241/44 or heartbeat field 21 to show it. It does not write when already set.
func (wb *Wallbox) SetPhaseMode(mode uint64, timeout time.Duration) error {
	// field 21 is the phase count in effect: 1 = one phase, 0 = three phases
	// (on hardware 2026-10-24 also after a written 3p)
	var effective uint64
	switch mode {
	case PhaseOne:
		effective = 1
	case PhaseThree:
		effective = 0
	default:
		return fmt.Errorf("invalid phase mode: %d", mode)
	}

	if wb.hasPhaseMode(mode) {
		wb.log.DEBUG.Printf("%s: phase mode %d already set, not sending", wb.serial, mode)
		return nil
	}

	return wb.write(fmt.Sprintf("phasemode=%d", mode), func(seq uint64) []byte {
		return BuildPhaseSet(wb.serial, wb.gateway, mode, seq)
	}, func(u Update) bool {
		switch {
		case u.Settings != nil:
			return u.Settings.PhaseModeRaw == mode
		case u.Heartbeat != nil:
			return u.Heartbeat.HasPhaseMode && u.Heartbeat.PhaseModeRaw == effective
		default:
			return false
		}
	}, timeout)
}

// hasPhaseMode reports whether fresh telemetry shows mode selected. Field 21 = 0
// is reported in auto and in 3p alike, so only a 241/44 report shows three phases.
func (wb *Wallbox) hasPhaseMode(mode uint64) bool {
	if s, ok := wb.latestSettings(); ok {
		return s.PhaseModeRaw == mode
	}

	hb, ok := wb.latest()
	if !ok || !hb.HasPhaseMode {
		return false
	}

	return mode == PhaseOne && hb.PhaseModeRaw == 1
}
