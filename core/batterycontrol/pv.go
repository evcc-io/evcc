// Package batterycontrol manages ownership-aware PV battery charging.
package batterycontrol

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
)

const (
	LeaseDuration = 90 * time.Second
	MaxSampleAge  = 10 * time.Second
	// Ignore the observed idle offset only when the device also denies charging.
	NativeChargeIdleTolerance = 5.0
)

// Phase is the controller's current state.
type Phase string

const (
	Observing  Phase = "observing"
	Native     Phase = "native-charging"
	Foreign    Phase = "foreign-or-unknown-control"
	Blocked    Phase = "safe-control-unavailable"
	Active     Phase = "active"
	Recovering Phase = "recovering"
)

// Input contains current site measurements, settings and arbitration outcome.
type Input struct {
	Valid, Allowed    bool
	Grid, PV, Reserve float64
	// UnavailablePower excludes other battery discharge and newly allocated loads
	// from the PV budget without mistaking those estimates for measured grid import.
	UnavailablePower     float64
	StartPower, MaxPower float64
	MaxSoc               float64
}

// Session only owns a device-verified lease created during this process lifetime.
// It is deliberately not restored from a persisted boolean or previous setpoint.
type Session struct {
	Phase                  Phase
	Reason                 string
	ObservedMode           api.BatteryOperatingMode
	ObservedNativeMode     string
	ObservedPower          float64
	ObservedNativeCharging bool
	leaseID                string
	power                  float64
	now                    func() time.Time
}

// Power returns the last confirmed requested power while active.
func (s *Session) Power() float64 {
	if s.Phase == Active {
		return s.power
	}
	return 0
}

func nativeCharging(state api.BatteryControlState) bool {
	return state.NativeCharging || state.Power < -NativeChargeIdleTolerance
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func (s *Session) transition(phase Phase, reason string) {
	s.Phase, s.Reason = phase, reason
}

func fresh(state api.BatteryControlState, now time.Time) bool {
	return !state.ObservedAt.IsZero() && !state.ObservedAt.After(now) && now.Sub(state.ObservedAt) <= MaxSampleAge &&
		finite(state.Power) && finite(state.Soc) && state.Soc >= 0 && state.Soc <= 100
}

// Stop releases only this session's fenced lease. Failure blocks further charging.
func (s *Session) Stop(ctrl api.BatteryPVLeaseController, reason string) error {
	if s.leaseID == "" {
		return nil
	}
	s.transition(Recovering, reason)
	if ctrl == nil {
		return fmt.Errorf("owned battery controller unavailable")
	}
	if err := ctrl.ReleaseBatteryPVControl(s.leaseID); err != nil {
		return err
	}
	s.leaseID, s.power = "", 0
	s.transition(Observing, "control released")
	return nil
}

// Step observes before deciding and never falls back to an unfenced mode write.
func (s *Session) Step(reader api.BatteryControlStateReader, ctrl api.BatteryPVLeaseController, in Input) error {
	clock := s.now
	if clock == nil {
		clock = time.Now
	}
	if s.Phase == Recovering {
		return s.Stop(ctrl, "retrying fenced release")
	}
	s.ObservedMode, s.ObservedNativeMode, s.ObservedPower = api.BatteryOperatingUnknown, "", 0
	if reader == nil {
		s.transition(Blocked, "actual battery mode unavailable")
		return s.Stop(ctrl, s.Reason)
	}
	state, err := reader.BatteryControlState()
	now := clock()
	if err != nil || !fresh(state, now) {
		s.transition(Blocked, "battery observation invalid or stale")
		stopErr := s.Stop(ctrl, s.Reason)
		if err != nil {
			return errors.Join(fmt.Errorf("battery observation: %w", err), stopErr)
		}
		return stopErr
	}
	s.ObservedMode, s.ObservedNativeMode, s.ObservedPower = state.Mode, state.NativeMode, state.Power
	s.ObservedNativeCharging = state.NativeCharging
	owned := s.leaseID != "" && state.LeaseID == s.leaseID && state.LeaseUntil.After(now) && state.Mode == api.BatteryOperatingManual
	if s.leaseID != "" && !owned {
		// Release is conditional on the old ID, so it cannot modify a new owner's mode.
		if err := s.Stop(ctrl, "ownership lost"); err != nil {
			return err
		}
		s.transition(Foreign, "ownership lost; no automatic reacquisition this cycle")
		return nil
	}
	if !owned {
		if state.Mode != api.BatteryOperatingAuto || state.LeaseID != "" {
			s.transition(Foreign, "manual or unknown mode without this process's lease")
			return nil
		}
		if nativeCharging(state) {
			s.transition(Native, "battery is charging autonomously")
			return nil
		}
		if ctrl == nil {
			s.transition(Blocked, "device has no verified ownership and expiry mechanism")
			return nil
		}
	}
	valid := in.Valid && finite(in.Grid) && finite(in.PV) && in.PV > 0 && finite(in.Reserve) && in.Reserve >= 0 &&
		finite(in.UnavailablePower) && in.UnavailablePower >= 0 &&
		finite(in.StartPower) && in.StartPower > 0 && finite(in.MaxPower) && in.MaxPower > 0 &&
		finite(in.MaxSoc) && in.MaxSoc > 0 && in.MaxSoc <= 100
	if !valid || !in.Allowed || state.Soc >= in.MaxSoc {
		if owned {
			return s.Stop(ctrl, "no safe charging budget or higher-priority control")
		}
		s.transition(Observing, "no safe charging budget or higher-priority control")
		return nil
	}
	if owned && in.Grid > 0 {
		// The start threshold is not a stop threshold. Actual grid import is:
		// release now, rather than continuing forced charging at a lower setpoint.
		return s.Stop(ctrl, "grid import detected")
	}
	charge := 0.0
	if owned {
		charge = max(0, -state.Power)
	}
	budgetGrid := in.Grid + in.UnavailablePower
	export := -budgetGrid - max(0, state.Power)
	if !owned && export < in.StartPower {
		s.transition(Observing, "below start threshold")
		return nil
	}
	target := math.Floor(min(in.MaxPower, in.PV, max(0, charge+export-in.Reserve)))
	if owned && budgetGrid <= 0 && budgetGrid >= -in.Reserve {
		// Keep the setting around zero/export reserve instead of integrating measurement noise.
		target = min(s.power, in.MaxPower, in.PV)
	}
	if target <= 0 {
		if owned {
			return s.Stop(ctrl, "no usable PV surplus")
		}
		s.transition(Observing, "no usable PV surplus")
		return nil
	}
	if !owned {
		s.leaseID = rand.Text()
		err = ctrl.AcquireBatteryPVControl(s.leaseID, target, LeaseDuration)
	} else {
		err = ctrl.RenewBatteryPVControl(s.leaseID, target, LeaseDuration)
	}
	if err != nil {
		stopErr := s.Stop(ctrl, "command failed; fenced release required")
		return errors.Join(fmt.Errorf("battery PV command: %w", err), stopErr)
	}
	confirmed, err := reader.BatteryControlState()
	if err != nil || confirmed.Mode != api.BatteryOperatingManual || confirmed.LeaseID != s.leaseID ||
		!confirmed.LeaseUntil.After(clock()) || !fresh(confirmed, clock()) {
		stopErr := s.Stop(ctrl, "ownership acknowledgement missing")
		return errors.Join(fmt.Errorf("battery PV ownership not confirmed"), err, stopErr)
	}
	s.power = target
	s.transition(Active, "device-confirmed expiring lease")
	return nil
}
