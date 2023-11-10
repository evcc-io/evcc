package api

// BatteryMode is the home battery operation mode. Valid values are normal, locked and charge
type BatteryMode int

//go:generate enumer -type BatteryMode
const (
	BatteryUnknown BatteryMode = iota
	BatteryNormal
	BatteryLocked
	BatteryCharge
)
