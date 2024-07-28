package ocpp

const (
	// Core profile keys
	KeyGetConfigurationMaxKeys           = "GetConfigurationMaxKeys"
	KeyNumberOfConnectors                = "NumberOfConnectors"
	KeySupportedFeatureProfiles          = "SupportedFeatureProfiles"
	KeyWebSocketPingInterval             = "WebSocketPingInterval"
	KeyStopTransactionOnEVSideDisconnect = "StopTransactionOnEVSideDisconnect"

	KeyMeterValuesSampledData          = "MeterValuesSampledData"
	KeyMeterValuesSampledDataMaxLength = "MeterValuesSampledDataMaxLength"
	KeyMeterValueSampleInterval        = "MeterValueSampleInterval"
	KeyMeterValuesAlignedData          = "MeterValuesAlignedData"
	KeyMeterValuesAlignedDataMaxLength = "MeterValuesAlignedDataMaxLength"
	KeyClockAlignedDataInterval        = "ClockAlignedDataInterval"
	KeyStopTxnSampledData              = "StopTxnSampledData"
	KeyStopTxnSampledDataMaxLength     = "StopTxnSampledDataMaxLength"
	KeyStopTxnAlignedData              = "StopTxnAlignedData"
	KeyStopTxnAlignedDataMaxLength     = "StopTxnAlignedDataMaxLength"

	// SmartCharging profile keys
	KeyChargeProfileMaxStackLevel              = "ChargeProfileMaxStackLevel"
	KeyChargingScheduleAllowedChargingRateUnit = "ChargingScheduleAllowedChargingRateUnit"
	KeyChargingScheduleMaxPeriods              = "ChargingScheduleMaxPeriods"
	KeyConnectorSwitch3to1PhaseSupported       = "ConnectorSwitch3to1PhaseSupported"
	KeyMaxChargingProfilesInstalled            = "MaxChargingProfilesInstalled"

	// Vendor specific keys
	KeyAlfenPlugAndChargeIdentifier = "PlugAndChargeIdentifier"
)
