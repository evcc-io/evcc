package ocpp

import "time"

const (
	Timeout = 30 * time.Second // default request / response timeout on protocol level

	// Core profile keys
	KeyMeterValueSampleInterval        = "MeterValueSampleInterval"
	KeyMeterValuesSampledData          = "MeterValuesSampledData"
	KeyMeterValuesSampledDataMaxLength = "MeterValuesSampledDataMaxLength"
	KeyNumberOfConnectors              = "NumberOfConnectors"
	KeySupportedFeatureProfiles        = "SupportedFeatureProfiles"
	KeyWebSocketPingInterval           = "WebSocketPingInterval"

	// SmartCharging profile keys
	KeyChargeProfileMaxStackLevel              = "ChargeProfileMaxStackLevel"
	KeyChargingScheduleAllowedChargingRateUnit = "ChargingScheduleAllowedChargingRateUnit"
	KeyConnectorSwitch3to1PhaseSupported       = "ConnectorSwitch3to1PhaseSupported"
	KeyMaxChargingProfilesInstalled            = "MaxChargingProfilesInstalled"

	// Vendor specific keys
	KeyAlfenPlugAndChargeIdentifier = "PlugAndChargeIdentifier"
	KeyEvBoxSupportedMeasurands     = "evb_SupportedMeasurands"
)
