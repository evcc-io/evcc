package ocpp

import "time"

var Timeout = time.Minute // default request / response timeout on protocol level

const (
	// HeartbeatInterval is the interval requested from the charge point in BootNotification
	HeartbeatInterval = time.Minute

	// PingWait is the websocket inactivity timeout. Must exceed HeartbeatInterval,
	// otherwise chargers that don't send websocket pings are disconnected while idle.
	PingWait = 3 * HeartbeatInterval
)

// TriggerBootDelay defines how long to wait after WebSocket connect before
// proactively triggering a BootNotification. This allows the connection to
// stabilize and gives the charger a chance to send a spontaneous BootNotification.
// It is a var so tests can shorten it.
var TriggerBootDelay = 5 * time.Second

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
