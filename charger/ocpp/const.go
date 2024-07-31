package ocpp

const (
	// Core profile keys
	KeyGetConfigurationMaxKeys           = "GetConfigurationMaxKeys"
	KeyMeterValueSampleInterval          = "MeterValueSampleInterval"
	KeyMeterValuesSampledData            = "MeterValuesSampledData"
	KeyMeterValuesSampledDataMaxLength   = "MeterValuesSampledDataMaxLength"
	KeyNumberOfConnectors                = "NumberOfConnectors"
	KeyStopTransactionOnEVSideDisconnect = "StopTransactionOnEVSideDisconnect"
	KeySupportedFeatureProfiles          = "SupportedFeatureProfiles"
	KeyWebSocketPingInterval             = "WebSocketPingInterval"

	// SmartCharging profile keys
	KeyChargeProfileMaxStackLevel              = "ChargeProfileMaxStackLevel"
	KeyChargingScheduleAllowedChargingRateUnit = "ChargingScheduleAllowedChargingRateUnit"
	KeyChargingScheduleMaxPeriods              = "ChargingScheduleMaxPeriods"
	KeyConnectorSwitch3to1PhaseSupported       = "ConnectorSwitch3to1PhaseSupported"
	KeyMaxChargingProfilesInstalled            = "MaxChargingProfilesInstalled"

	// Vendor specific keys
	KeyAlfenPlugAndChargeIdentifier = "PlugAndChargeIdentifier"
)
