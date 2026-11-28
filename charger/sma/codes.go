// Package sma contains channels and codes shared by all SMA EV Charger and eCharger protocols
package sma

import (
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/api"
)

// Channel ids as used by the REST and SEMP parameter interfaces
const (
	OperationEVehChaStt = "Measurement.Operation.EVeh.ChaStt"
	ChaSessWhIn         = "Measurement.ChaSess.WhIn"
	ChrgActChaMod       = "Parameter.Chrg.ActChaMod"
	ChrgChrgApv         = "Parameter.Chrg.ChrgApv"
	ChrgChrgLok         = "Parameter.Chrg.ChrgLok"
	ChrgMinSwTm         = "Parameter.Chrg.MinSwTm"
	ChrgStpWhenFl       = "Parameter.Chrg.StpWhenFl"
	ChrgUseEnergyMeter  = "Parameter.Chrg.UseEnergyMeter"
	InverterWMaxIn      = "Parameter.Inverter.WMaxIn"
	InverterTmsOutPrc   = "Parameter.Inverter.CtlComCfg.WCtlCom.TmsOutPrc"
	SetpointWSptMax     = "Setpoint.PlantControl.Inverter.WModCfg.WCtlComCfg.WSptMax"
	SetpointWSptMin     = "Setpoint.PlantControl.Inverter.WModCfg.WCtlComCfg.WSptMin"
)

// The charger selects 3p from about 4350 W setpoint power and 1p below without hysteresis, setpoints keep a margin
const (
	PhaseThreshold = 4350 // W
	MinPower3p     = 4500 // W
	MaxPower1p     = 4300 // W
)

// Phases returns the phases the charger selects for the setpoint power
func Phases(power float64) int {
	if power >= PhaseThreshold {
		return 3
	}
	return 1
}

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

// Generic values, e.g. Parameter.Chrg.ChrgLok (access management) and Parameter.Chrg.StpWhenFl
const (
	Yes Code = 1129
	No  Code = 1130
)

// Measurement.Operation.EVeh.ChaStt
const (
	Warning             Code = 200106 // Warnung
	NotConnected        Code = 200111 // nicht verbunden
	SleepMode           Code = 200112 // schlafend
	ActiveMode          Code = 200113 // Laden aktiv
	ChargeTargetReached Code = 200114 // Ladeziel erreicht
	StationFault        Code = 200115 // Fehler
	StationLocked       Code = 5169   // gesperrt, connected and waiting for charging release
)

// Parameter.Chrg.ActChaMod
const (
	BoostCharging     Code = 4718 // Schnellladen
	OptimizedCharging Code = 4719 // Laden mit PV-Überschuss
	SetpointCharging  Code = 4720 // Laden mit Zielvorgabe
	ChargeStop        Code = 4721 // Ladestopp
)

// Parameter.Chrg.ChrgApv
const (
	ChargingLock    Code = 5171 // Ladesperre
	ChargingRelease Code = 5172 // Ladefreigabe
)

// Parameter.Chrg.MinSwTm
const (
	SwitchingTime120s Code = 6715 // 120 s
	SwitchingTime90s  Code = 6716 // 90 s
	SwitchingTime60s  Code = 6717 // 60 s
	SwitchingTime30s  Code = 6718 // 30 s
	SwitchingTime15s  Code = 6719 // 15 s
)

// ChargeStatus maps the charging status
func ChargeStatus(c Code) (api.ChargeStatus, error) {
	switch c {
	case NotConnected:
		return api.StatusA, nil
	case SleepMode, ChargeTargetReached, StationLocked:
		return api.StatusB, nil
	case ActiveMode:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", c)
	}
}

// StatusReason reports waiting for authorization while access management (ChrgLok) requires a release (ChrgApv) not yet given
func StatusReason(chrgLok, chrgApv Code) api.Reason {
	if chrgLok == Yes && chrgApv != ChargingRelease {
		return api.ReasonWaitingForAuthorization
	}
	return api.ReasonUnknown
}
