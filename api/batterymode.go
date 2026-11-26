package api

// BatteryMode is the home battery operation mode. Valid values are normal, holddischarge, charge, holdcharge and discharge
type BatteryMode int

//go:generate go tool enumer -type BatteryMode -trimprefix Battery -transform=lower
const (
	BatteryUnknown BatteryMode = iota
	BatteryNormal
	BatteryHoldDischarge
	BatteryCharge
	BatteryHoldCharge
	BatteryDischarge // forced discharge to grid (feed-in arbitrage)
)
