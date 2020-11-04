package openwb

// predefined openWB topic names
const (
	// configured
	ConfiguredTopic             = "boolChargePointConfigured"
	HouseBatteryConfiguredTopic = "boolHouseBatteryConfigured"

	// status
	PluggedTopic  = "boolPlugStat"
	ChargingTopic = "boolChargeStat"

	// getter/setter
	EnabledTopic    = "ChargePointEnabled"
	MaxCurrentTopic = "DirectChargeAmps"

	// charge power
	ChargePowerTopic       = "W"
	ChargeTotalEnergyTopic = "kWhCounter"

	// general measurements
	PowerTopic   = "W"
	CurrentTopic = "APhase" // 1..3
)
