package ocpp

const (
	// Core profile keys
	KeyNumberOfConnectors = "NumberOfConnectors"

	// Meter profile keys
	KeyMeterValuesSampledData   = "MeterValuesSampledData"
	KeyMeterValueSampleInterval = "MeterValueSampleInterval"

	// Smart Charging profile keys
	KeyChargeProfileMaxStackLevel              = "ChargeProfileMaxStackLevel"
	KeyChargingScheduleAllowedChargingRateUnit = "ChargingScheduleAllowedChargingRateUnit"
	KeyChargingScheduleMaxPeriods              = "ChargingScheduleMaxPeriods"
	KeyConnectorSwitch3to1PhaseSupported       = "ConnectorSwitch3to1PhaseSupported"
	KeyMaxChargingProfilesInstalled            = "MaxChargingProfilesInstalled"

	// Alfen specific keys
	KeyAlfenPlugAndChargeIdentifier = "PlugAndChargeIdentifier"
)
