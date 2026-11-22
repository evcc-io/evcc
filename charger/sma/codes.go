// Package sma contains channels and codes shared by all SMA EV Charger and eCharger protocols
package sma

import (
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/api"
)

// Channel ids as used by the REST and SEMP parameter interfaces
const (
	ChannelStatus           = "Measurement.Operation.EVeh.ChaStt"
	ChannelSessionEnergy    = "Measurement.ChaSess.WhIn"
	ChannelMode             = "Parameter.Chrg.ActChaMod"
	ChannelRelease          = "Parameter.Chrg.ChrgApv"
	ChannelAccessManagement = "Parameter.Chrg.ChrgLok"
)

// Code is an SMA enum value
type Code uint32

// String returns the decimal representation used by the REST and SEMP interfaces
func (c Code) String() string {
	return strconv.FormatUint(uint64(c), 10)
}

// ParseCode parses the decimal representation used by the REST and SEMP interfaces
func ParseCode(s string) (Code, error) {
	c, err := strconv.ParseUint(s, 10, 32)
	return Code(c), err
}

// Measurement.Operation.EVeh.ChaStt
const (
	StatusWarning      Code = 200106
	StatusNotConnected Code = 200111
	StatusSleeping     Code = 200112
	StatusCharging     Code = 200113
	StatusFinished     Code = 200114 // charge target reached
	StatusError        Code = 200115
	StatusLocked       Code = 5169 // connected, waiting for charging release
)

// Parameter.Chrg.ActChaMod
const (
	ModeFast      Code = 4718
	ModePVSurplus Code = 4719
	ModeTarget    Code = 4720
	ModeStop      Code = 4721
)

// Parameter.Chrg.ChrgApv
const (
	ChargingLocked   Code = 5171
	ChargingReleased Code = 5172
)

// Parameter.Chrg.ChrgLok
const (
	AccessManagementOn  Code = 1129 // charging release via app or RFID required
	AccessManagementOff Code = 1130
)

// ChargeStatus maps the charging status
func ChargeStatus(c Code) (api.ChargeStatus, error) {
	switch c {
	case StatusNotConnected:
		return api.StatusA, nil
	case StatusSleeping, StatusFinished, StatusLocked:
		return api.StatusB, nil
	case StatusCharging:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", c)
	}
}

// StatusReason reports waiting for authorization while locked by access management
func StatusReason(status Code, accessManagement bool) api.Reason {
	if status == StatusLocked && accessManagement {
		return api.ReasonWaitingForAuthorization
	}
	return api.ReasonUnknown
}
