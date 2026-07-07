package ocpp

import "time"

var Timeout = time.Minute // default request / response timeout on protocol level

// TriggerBootDelay defines how long to wait after WebSocket connect before
// proactively triggering a BootNotification. This allows the connection to
// stabilize and gives the charger a chance to send a spontaneous BootNotification.
<<<<<<< HEAD
const TriggerBootDelay = 5 * time.Second
=======
// It is a var so tests can shorten it.
var TriggerBootDelay = 5 * time.Second
>>>>>>> origin/master

const (
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
	KeyAlfenPlugAndChargeIdentifier      = "PlugAndChargeIdentifier"
	KeyChargeAmpsPhaseSwitchingSupported = "ACPhaseSwitchingSupported"
	KeyEvBoxSupportedMeasurands          = "evb_SupportedMeasurands"
)
