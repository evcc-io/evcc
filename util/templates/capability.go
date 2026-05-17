package templates

type Capability int

//go:generate go tool enumer -type Capability -linecomment -text
const (
	_ Capability = iota
	// ISO 15118-2 support
	CapabilityISO151182 // iso151182
	// granular current control support
	CapabilityMilliAmps // mA
	// RFID support
	CapabilityRFID // rfid
	// 1P/3P phase switching support
	Capability1p3p // 1p3p
	// battery control support
	CapabilityBatteryControl // battery-control
	// built-in energy meter support
	CapabilityMeter // meter
)
