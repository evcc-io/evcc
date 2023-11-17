package site

// BatteryControl is the battery control mode bit field
type BatteryControl int

//go:generate enumer -type BatteryControl -trimprefix BatteryControl -transform=lower
const (
	BatteryControlDisabled BatteryControl = iota
	BatteryControlDischarge
	BatteryControlCharge
	BatteryControlEnabled
)
