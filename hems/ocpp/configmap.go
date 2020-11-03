package ocpp

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

func getDefaultConfig() ConfigMap {
	intBase := 10

	cfg := map[string]core.ConfigurationKey{
		SupportedFeatureProfiles:                {Key: SupportedFeatureProfiles, Readonly: true, Value: core.ProfileName},
		AuthorizeRemoteTxRequests:               {Key: AuthorizeRemoteTxRequests, Readonly: true, Value: strconv.FormatBool(false)},
		ClockAlignedDataInterval:                {Key: ClockAlignedDataInterval, Readonly: false, Value: strconv.FormatInt(0, intBase)},
		ConnectionTimeOut:                       {Key: ConnectionTimeOut, Readonly: false, Value: strconv.FormatInt(60, intBase)},
		ConnectorPhaseRotation:                  {Key: ConnectorPhaseRotation, Readonly: false, Value: "Unknown"},
		GetConfigurationMaxKeys:                 {Key: GetConfigurationMaxKeys, Readonly: true, Value: strconv.FormatInt(50, intBase)},
		HeartbeatInterval:                       {Key: HeartbeatInterval, Readonly: false, Value: strconv.FormatInt(86400, intBase)},
		LocalAuthorizeOffline:                   {Key: LocalAuthorizeOffline, Readonly: false, Value: strconv.FormatBool(true)},
		LocalPreAuthorize:                       {Key: LocalPreAuthorize, Readonly: false, Value: strconv.FormatBool(false)},
		MeterValuesAlignedData:                  {Key: MeterValuesAlignedData, Readonly: false, Value: string(types.MeasurandEnergyActiveExportRegister)},
		MeterValuesSampledData:                  {Key: MeterValuesSampledData, Readonly: false, Value: string(types.MeasurandEnergyActiveExportRegister)},
		MeterValueSampleInterval:                {Key: MeterValueSampleInterval, Readonly: false, Value: strconv.FormatInt(5, intBase)},
		NumberOfConnectors:                      {Key: NumberOfConnectors, Readonly: true, Value: strconv.FormatInt(1, intBase)},
		ResetRetries:                            {Key: ResetRetries, Readonly: false, Value: strconv.FormatInt(10, intBase)},
		StopTransactionOnEVSideDisconnect:       {Key: StopTransactionOnEVSideDisconnect, Readonly: false, Value: strconv.FormatBool(true)},
		StopTransactionOnInvalidID:              {Key: StopTransactionOnInvalidId, Readonly: false, Value: strconv.FormatBool(true)},
		StopTxnAlignedData:                      {Key: StopTxnAlignedData, Readonly: false, Value: strconv.FormatBool(true)},
		StopTxnSampledData:                      {Key: StopTxnSampledData, Readonly: false, Value: string(types.MeasurandEnergyActiveExportRegister)},
		TransactionMessageAttempts:              {Key: TransactionMessageAttempts, Readonly: false, Value: strconv.FormatInt(5, intBase)},
		TransactionMessageRetryInterval:         {Key: TransactionMessageRetryInterval, Readonly: false, Value: strconv.FormatInt(60, intBase)},
		UnlockConnectorOnEVSideDisconnect:       {Key: UnlockConnectorOnEVSideDisconnect, Readonly: false, Value: strconv.FormatBool(true)},
		WebSocketPingInterval:                   {Key: WebSocketPingInterval, Readonly: false, Value: strconv.FormatInt(54, intBase)},
		LocalAuthListEnabled:                    {Key: LocalAuthListEnabled, Readonly: false, Value: strconv.FormatBool(true)},
		LocalAuthListMaxLength:                  {Key: LocalAuthListMaxLength, Readonly: true, Value: strconv.FormatInt(100, intBase)},
		SendLocalListMaxLength:                  {Key: SendLocalListMaxLength, Readonly: true, Value: strconv.FormatInt(20, intBase)},
		ChargeProfileMaxStackLevel:              {Key: ChargeProfileMaxStackLevel, Readonly: true, Value: strconv.FormatInt(10, intBase)},
		ChargingScheduleAllowedChargingRateUnit: {Key: ChargingScheduleAllowedChargingRateUnit, Readonly: true, Value: "Power"},
		ChargingScheduleMaxPeriods:              {Key: ChargingScheduleMaxPeriods, Readonly: true, Value: strconv.FormatInt(5, intBase)},
		MaxChargingProfilesInstalled:            {Key: MaxChargingProfilesInstalled, Readonly: true, Value: strconv.FormatInt(10, intBase)},
	}

	return cfg
}
