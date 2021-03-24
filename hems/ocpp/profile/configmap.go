package profile

import (
	"strconv"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// modified from https://github.com/lorenzodonini/ocpp-go/tree/master/example/1.6/cp

// configuration options
const (
	AuthorizeRemoteTxRequests               string = "AuthorizeRemoteTxRequests"
	ClockAlignedDataInterval                string = "ClockAlignedDataInterval"
	ConnectionTimeOut                       string = "ConnectionTimeOut"
	ConnectorPhaseRotation                  string = "ConnectorPhaseRotation"
	GetConfigurationMaxKeys                 string = "GetConfigurationMaxKeys"
	HeartbeatInterval                       string = "HeartbeatInterval"
	LocalAuthorizeOffline                   string = "LocalAuthorizeOffline"
	LocalPreAuthorize                       string = "LocalPreAuthorize"
	MeterValuesAlignedData                  string = "MeterValuesAlignedData"
	MeterValuesSampledData                  string = "MeterValuesSampledData"
	MeterValueSampleInterval                string = "MeterValueSampleInterval"
	NumberOfConnectors                      string = "NumberOfConnectors"
	ResetRetries                            string = "ResetRetries"
	StopTransactionOnEVSideDisconnect       string = "StopTransactionOnEVSideDisconnect"
	StopTransactionOnInvalidID              string = "StopTransactionOnInvalidId"
	StopTxnAlignedData                      string = "StopTxnAlignedData"
	StopTxnSampledData                      string = "StopTxnSampledData"
	SupportedFeatureProfiles                string = "SupportedFeatureProfiles"
	TransactionMessageAttempts              string = "TransactionMessageAttempts"
	TransactionMessageRetryInterval         string = "TransactionMessageRetryInterval"
	UnlockConnectorOnEVSideDisconnect       string = "UnlockConnectorOnEVSideDisconnect"
	WebSocketPingInterval                   string = "WebSocketPingInterval"
	LocalAuthListEnabled                    string = "LocalAuthListEnabled"
	LocalAuthListMaxLength                  string = "LocalAuthListMaxLength"
	SendLocalListMaxLength                  string = "SendLocalListMaxLength"
	ChargeProfileMaxStackLevel              string = "ChargeProfileMaxStackLevel"
	ChargingScheduleAllowedChargingRateUnit string = "ChargingScheduleAllowedChargingRateUnit"
	ChargingScheduleMaxPeriods              string = "ChargingScheduleMaxPeriods"
	MaxChargingProfilesInstalled            string = "MaxChargingProfilesInstalled"
)

// ConfigMap defines the active configuration settings
type ConfigMap map[string]core.ConfigurationKey

// func (c ConfigMap) updateInt(key string, i int64) {
// 	configKey, ok := c[key]
// 	if ok {
// 		configKey.Value = strconv.FormatInt(i, 10)
// 	}
// 	c[key] = configKey
// }

// func (c ConfigMap) updateBool(key string, b bool) {
// 	configKey, ok := c[key]
// 	if ok {
// 		configKey.Value = strconv.FormatBool(b)
// 	}
// 	c[key] = configKey
// }

// func (c ConfigMap) getInt(key string) (int, bool) {
// 	configKey, ok := c[key]
// 	if !ok {
// 		return 0, ok
// 	}
// 	result, err := strconv.ParseInt(configKey.Value, 10, 32)
// 	if err != nil {
// 		return 0, false
// 	}
// 	return int(result), true
// }

// func (c ConfigMap) getBool(key string) (bool, bool) {
// 	configKey, ok := c[key]
// 	if !ok {
// 		return false, ok
// 	}
// 	result, err := strconv.ParseBool(configKey.Value)
// 	if err != nil {
// 		return false, false
// 	}
// 	return result, true
// }

func (c ConfigMap) set(key string, readonly bool, value string) {
	c[key] = core.ConfigurationKey{
		Key:      key,
		Readonly: readonly,
		Value:    value,
	}
}

func GetDefaultConfig() ConfigMap {
	intBase := 10

	var cfg ConfigMap = make(map[string]core.ConfigurationKey)

	// readonly
	cfg.set(SupportedFeatureProfiles, true, core.ProfileName)
	cfg.set(AuthorizeRemoteTxRequests, true, strconv.FormatBool(false))
	cfg.set(GetConfigurationMaxKeys, true, strconv.FormatInt(50, intBase))
	cfg.set(NumberOfConnectors, true, strconv.FormatInt(1, intBase))
	cfg.set(LocalAuthListMaxLength, true, strconv.FormatInt(100, intBase))
	cfg.set(SendLocalListMaxLength, true, strconv.FormatInt(20, intBase))
	cfg.set(ChargeProfileMaxStackLevel, true, strconv.FormatInt(10, intBase))
	cfg.set(ChargingScheduleAllowedChargingRateUnit, true, "Power")
	cfg.set(ChargingScheduleMaxPeriods, true, strconv.FormatInt(5, intBase))
	cfg.set(MaxChargingProfilesInstalled, true, strconv.FormatInt(10, intBase))

	// read/write
	cfg.set(ClockAlignedDataInterval, false, strconv.FormatInt(0, intBase))
	cfg.set(ConnectionTimeOut, false, strconv.FormatInt(60, intBase))
	cfg.set(ConnectorPhaseRotation, false, "Unknown")
	cfg.set(HeartbeatInterval, false, strconv.FormatInt(86400, intBase))
	cfg.set(LocalAuthorizeOffline, false, strconv.FormatBool(true))
	cfg.set(LocalAuthListEnabled, false, strconv.FormatBool(true))
	cfg.set(LocalPreAuthorize, false, strconv.FormatBool(false))
	cfg.set(MeterValuesAlignedData, false, string(types.MeasurandEnergyActiveExportRegister))
	cfg.set(MeterValuesSampledData, false, string(types.MeasurandEnergyActiveExportRegister))
	cfg.set(MeterValueSampleInterval, false, strconv.FormatInt(5, intBase))
	cfg.set(ResetRetries, false, strconv.FormatInt(10, intBase))
	cfg.set(StopTransactionOnEVSideDisconnect, false, strconv.FormatBool(true))
	cfg.set(StopTransactionOnInvalidID, false, strconv.FormatBool(true))
	cfg.set(StopTxnAlignedData, false, strconv.FormatBool(true))
	cfg.set(StopTxnSampledData, false, string(types.MeasurandEnergyActiveExportRegister))
	cfg.set(TransactionMessageAttempts, false, strconv.FormatInt(5, intBase))
	cfg.set(TransactionMessageRetryInterval, false, strconv.FormatInt(60, intBase))
	cfg.set(UnlockConnectorOnEVSideDisconnect, false, strconv.FormatBool(true))
	cfg.set(WebSocketPingInterval, false, strconv.FormatInt(54, intBase))

	return cfg
}
