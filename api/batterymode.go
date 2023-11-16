package api

// BatteryMode is the home battery operation mode. Valid values are normal, locked and charge
type BatteryMode int

//go:generate enumer -type BatteryMode -trimprefix Battery -transform=lower
const (
	BatteryUnknown BatteryMode = iota
	BatteryNormal
	BatteryHold
	BatteryCharge
)
