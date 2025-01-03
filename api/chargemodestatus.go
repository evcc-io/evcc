package api

import (
	"fmt"
	"strings"
)

// ChargeMode is the charge operation mode. Valid values are off, now, minpv and pv
type ChargeMode string

// Charge modes
const (
	ModeEmpty ChargeMode = ""
	ModeOff   ChargeMode = "off"
	ModeNow   ChargeMode = "now"
	ModeMinPV ChargeMode = "minpv"
	ModePV    ChargeMode = "pv"
)

// String implements Stringer
func (c ChargeMode) String() string {
	return string(c)
}

// ChargeStatus is the EV's charging status from A to F
type ChargeStatus string

// Charging states
const (
	StatusUnknown      ChargeStatus = ""
	StatusDisconnected ChargeStatus = "A" // No vehicle connected
	StatusConnected    ChargeStatus = "B" // Vehicle connected, no charging
	StatusCharging     ChargeStatus = "C" // Vehicle charging
)

// ChargeStatusString converts from IEC 62196 string to ChargeStatus
func ChargeStatusString(status string) (ChargeStatus, error) {
	s := strings.ToUpper(strings.Trim(status, "\x00 "))

	if len(s) == 0 {
		return StatusUnknown, fmt.Errorf("invalid status: %s", status)
	}

	switch s1 := s[:1]; s1 {
	case "A":
		return StatusDisconnected, nil

	case "B":
		return StatusConnected, nil

	case "C", "D":
		if s == "C1" || s == "D1" {
			return StatusConnected, nil
		}
		return StatusCharging, nil

	case "E", "F":
		return StatusUnknown, fmt.Errorf("error status: %s", s)

	default:
		return StatusUnknown, fmt.Errorf("invalid status: %s", status)
	}
}

// String implements Stringer
func (c ChargeStatus) String() string {
	return string(c)
}
