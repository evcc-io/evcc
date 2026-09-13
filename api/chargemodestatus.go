package api

import (
	"fmt"
	"strings"
)

// ChargeMode is the charge operation mode. Valid values are off, smart and now
type ChargeMode string

// Charge modes
const (
	ModeEmpty ChargeMode = ""
	ModeOff   ChargeMode = "off"
	ModeNow   ChargeMode = "now"
	ModeSmart ChargeMode = "smart"

	// deprecated, accepted on input only, normalized to smart in SetMode
	ModeMinPV ChargeMode = "minpv"
	ModePV    ChargeMode = "pv"
)

// String implements Stringer
func (c ChargeMode) String() string {
	return string(c)
}

// Normalize maps deprecated pv/minpv to smart and the equivalent always charge state
func (c ChargeMode) Normalize() (ChargeMode, AlwaysCharge) {
	switch c {
	case ModeMinPV:
		return ModeSmart, AlwaysChargeOn
	case ModePV:
		return ModeSmart, AlwaysChargeOff
	default:
		return c, ""
	}
}

// AlwaysCharge makes smart mode charge continuously at least at minimum power
type AlwaysCharge string

// AlwaysCharge states
const (
	AlwaysChargeOff  AlwaysCharge = "off"
	AlwaysChargeOn   AlwaysCharge = "on"
	AlwaysChargeOnce AlwaysCharge = "once" // resets on disconnect
)

// Active reports if always charge is currently in effect
func (a AlwaysCharge) Active() bool {
	return a == AlwaysChargeOn || a == AlwaysChargeOnce
}

// ChargeStatus is the EV's charging status from A to F
type ChargeStatus string

// Charging states
const (
	StatusNone ChargeStatus = ""
	StatusA    ChargeStatus = "A" // Fzg. angeschlossen: nein    Laden aktiv: nein    Ladestation betriebsbereit, Fahrzeug getrennt
	StatusB    ChargeStatus = "B" // Fzg. angeschlossen:   ja    Laden aktiv: nein    Fahrzeug verbunden, Netzspannung liegt nicht an
	StatusC    ChargeStatus = "C" // Fzg. angeschlossen:   ja    Laden aktiv:   ja    Fahrzeug lädt, Netzspannung liegt an
	statusE    ChargeStatus = "E" // Fzg. angeschlossen:   ja    Laden aktiv: nein    Fehler Fahrzeug / Kabel (CP-Kurzschluss, 0V)
)

var StatusEasA = map[ChargeStatus]ChargeStatus{statusE: StatusA}

// ChargeStatusString converts a string to ChargeStatus
func ChargeStatusString(status string) (ChargeStatus, error) {
	s := strings.ToUpper(strings.Trim(status, "\x00 "))

	if len(s) == 0 {
		return StatusNone, fmt.Errorf("invalid status: %s", status)
	}

	switch s1 := s[:1]; s1 {
	case "A", "B":
		return ChargeStatus(s1), nil

	case "C", "D":
		if s == "C1" || s == "D1" {
			return StatusB, nil
		}
		return StatusC, nil

	case "E", "F":
		return ChargeStatus(s1), fmt.Errorf("invalid status: %s", s)

	default:
		return StatusNone, fmt.Errorf("invalid status: %s", status)
	}
}

// ChargeStatusStringWithMapping converts a string to ChargeStatus. In case of error, mapping is applied.
func ChargeStatusStringWithMapping(s string, m map[ChargeStatus]ChargeStatus) (ChargeStatus, error) {
	status, err := ChargeStatusString(s)
	if mappedStatus, ok := m[status]; ok && err != nil {
		return mappedStatus, nil
	}
	return status, err
}

// String implements Stringer
func (c ChargeStatus) String() string {
	return string(c)
}
