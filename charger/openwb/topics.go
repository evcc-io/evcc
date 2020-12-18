package openwb

// predefined openWB topic names
const (
	// alive
	TimestampTopic = "Timestamp"

	// status
	PluggedTopic    = "boolPlugStat"
	ChargingTopic   = "boolChargeStat"
	ConfiguredTopic = "boolChargePointConfigured"

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
