//go:generate stringer -type=SolarChargingMode -output=./solar_charging_mode_string.go

package mennekes

type SolarChargingMode uint16

const (
	NotActive        SolarChargingMode = iota
	StandardMode                       // max. charging power
	SunshineMode                       // surplus power
	SunshinePlusMode                   // min A (default 6A) + surplus power
)
