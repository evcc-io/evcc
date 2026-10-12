package ocpp2

import "time"

// Timeout ist das Standard-Request/Response-Timeout auf Protokollebene.
var Timeout = time.Minute

const (
    // heartbeatInterval: angefordertes Heartbeat-Intervall in BootNotification-Response.
    heartbeatInterval = time.Minute

    // pingWait muss heartbeatInterval überschreiten, damit Charger die keine
    // WebSocket-Pings senden nicht getrennt werden.
    pingWait = 3 * heartbeatInterval

    // DefaultStackLevel für alle von evcc gesendeten ChargingProfiles.
    DefaultStackLevel = 1

    // DefaultChargingProfileId für TxDefault-Profile.
    DefaultChargingProfileId = 1
)

// TriggerBootDelay: Wartezeit nach WebSocket-Connect bevor BootNotification aktiv
// getriggert wird (gibt dem Charger Zeit, sie spontan zu senden).
var TriggerBootDelay = 5 * time.Second

// OCPP 2.0.1 Variable-Namen für GetVariables/SetVariables
// (ersetzt die GetConfiguration/ChangeConfiguration Keys aus OCPP 1.6)
const (
    // SampledDataCtrlr
    KeySampledDataTxUpdatedMeasurands = "SampledDataTxUpdatedMeasurands"
    KeySampledDataTxUpdatedInterval   = "SampledDataTxUpdatedInterval"
    KeySampledDataTxEndedMeasurands   = "SampledDataTxEndedMeasurands"

    // SmartChargingCtrlr
    KeySmartChargingCtrlrAvailable = "SmartChargingCtrlrAvailable"
    KeySmartChargingCtrlrEnabled   = "SmartChargingCtrlrEnabled"
)