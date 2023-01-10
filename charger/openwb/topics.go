package openwb

import "time"

// predefined openWB topic names
const (
	Timeout           = 15 * time.Second
	HeartbeatInterval = 10 * time.Second // loadpoint only client heartbeat

	// root topic
	RootTopic = "openWB"

	// alive
	TimestampTopic = "Timestamp"

	// status
	PluggedTopic    = "boolPlugStat"
	ChargingTopic   = "boolChargeStat"
	ConfiguredTopic = "boolChargePointConfigured"

	// getter/setter
	EnabledTopic    = "ChargePointEnabled"
	MaxCurrentTopic = "AConfigured" // was DirectChargeAmps
	PhasesTopic     = "U1p3p"
	RfidTopic       = "rfid"

	// charge power
	ChargePowerTopic       = "W"
	ChargeTotalEnergyTopic = "kWhCounter"

	// vehicle
	VehicleSoCTopic = "Soc"

	// general measurements
	PowerTopic   = "W"
	SoCTopic     = "%Soc"
	CurrentTopic = "APhase" // 1..3

	// configuration
	PvConfigured      = "boolPVConfigured"
	BatteryConfigured = "boolHouseBatteryConfigured"

	// loadpoint only topics

	// TODO cleanup after https://github.com/snaptec/openWB/issues/1757
	// openWB/set/isss/heartbeat
	// openWB/set/isss/ClearRfid
	// openWB/set/isss/Cpulp1
	// openWB/set/isss/Current
	// openWB/set/isss/Lp2Current
	// openWB/set/isss/U1p3p
	// openWB/set/isss/U1p3pLp2

	SlaveSetter = "set/isss"

	SlaveHeartbeatTopic     = "heartbeat"
	SlaveChargeCurrentTopic = "Current"
	SlavePhasesTopic        = "U1p3p"
	SlaveClearRfidTopic     = "ClearRfid"
	SlaveCPInterruptTopic   = "Cpulp1"
)
