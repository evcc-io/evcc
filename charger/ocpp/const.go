package ocpp

import "time"

var Timeout = time.Minute // default request / response timeout on protocol level

const (
	heartbeatInterval = time.Minute // heartbeat interval requested in BootNotification

	// pingWait must exceed heartbeatInterval, otherwise chargers not sending
	// websocket pings are disconnected while idle
	pingWait = 3 * heartbeatInterval
)

// TriggerBootDelay defines how long to wait after WebSocket connect before
// proactively triggering a BootNotification. This allows the connection to
// stabilize and gives the charger a chance to send a spontaneous BootNotification.
// It is a var so tests can shorten it.
var TriggerBootDelay = 5 * time.Second

// ChargingPowerThreshold is the minimum measured power (W) that counts as an
// EV actually drawing energy. It sits well above charger standby draw and well
// below the ~1.4kW a 6A single-phase charge pulls, so it reliably separates a
// live transaction from an idle one. Used to correct chargers (e.g. Grizzl-E)
// that keep reporting Suspended* while already delivering.
const ChargingPowerThreshold = 100 // W

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
