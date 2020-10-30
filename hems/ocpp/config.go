package ocpp

import (
	"fmt"
	"strconv"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

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
	StopTransactionOnInvalidId              string = "StopTransactionOnInvalidId"
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

type ConfigMap map[string]core.ConfigurationKey

func (c ConfigMap) updateInt(key string, i int64) {
	configKey, ok := c[key]
	if ok {
		configKey.Value = strconv.FormatInt(i, 10)
	}
	c[key] = configKey
}

func (c ConfigMap) updateBool(key string, b bool) {
	configKey, ok := c[key]
	if ok {
		configKey.Value = strconv.FormatBool(b)
	}
	c[key] = configKey
}

func (c ConfigMap) getInt(key string) (int, bool) {
	configKey, ok := c[key]
	if !ok {
		return 0, ok
	}
	result, err := strconv.ParseInt(configKey.Value, 10, 32)
	if err != nil {
		return 0, false
	}
	return int(result), true
}

func (c ConfigMap) getBool(key string) (bool, bool) {
	configKey, ok := c[key]
	if !ok {
		return false, ok
	}
	result, err := strconv.ParseBool(configKey.Value)
	if err != nil {
		return false, false
	}
	return result, true
}

func getDefaultConfig() ConfigMap {
	intBase := 10
	cfg := map[string]core.ConfigurationKey{}
	cfg[AuthorizeRemoteTxRequests] = core.ConfigurationKey{Key: AuthorizeRemoteTxRequests, Readonly: true, Value: strconv.FormatBool(false)}
	cfg[ClockAlignedDataInterval] = core.ConfigurationKey{Key: ClockAlignedDataInterval, Readonly: false, Value: strconv.FormatInt(0, intBase)}
	cfg[ConnectionTimeOut] = core.ConfigurationKey{Key: ConnectionTimeOut, Readonly: false, Value: strconv.FormatInt(60, intBase)}
	cfg[ConnectorPhaseRotation] = core.ConfigurationKey{Key: ConnectorPhaseRotation, Readonly: false, Value: "Unknown"}
	cfg[GetConfigurationMaxKeys] = core.ConfigurationKey{Key: GetConfigurationMaxKeys, Readonly: true, Value: strconv.FormatInt(50, intBase)}
	cfg[HeartbeatInterval] = core.ConfigurationKey{Key: HeartbeatInterval, Readonly: false, Value: strconv.FormatInt(86400, intBase)}
	cfg[LocalAuthorizeOffline] = core.ConfigurationKey{Key: LocalAuthorizeOffline, Readonly: false, Value: strconv.FormatBool(true)}
	cfg[LocalPreAuthorize] = core.ConfigurationKey{Key: LocalPreAuthorize, Readonly: false, Value: strconv.FormatBool(false)}
	cfg[MeterValuesAlignedData] = core.ConfigurationKey{Key: MeterValuesAlignedData, Readonly: false, Value: string(types.MeasurandEnergyActiveExportRegister)}
	cfg[MeterValuesSampledData] = core.ConfigurationKey{Key: MeterValuesSampledData, Readonly: false, Value: string(types.MeasurandEnergyActiveExportRegister)}
	cfg[MeterValueSampleInterval] = core.ConfigurationKey{Key: MeterValueSampleInterval, Readonly: false, Value: strconv.FormatInt(5, intBase)}
	cfg[NumberOfConnectors] = core.ConfigurationKey{Key: NumberOfConnectors, Readonly: true, Value: strconv.FormatInt(1, intBase)}
	cfg[ResetRetries] = core.ConfigurationKey{Key: ResetRetries, Readonly: false, Value: strconv.FormatInt(10, intBase)}
	cfg[StopTransactionOnEVSideDisconnect] = core.ConfigurationKey{Key: StopTransactionOnEVSideDisconnect, Readonly: false, Value: strconv.FormatBool(true)}
	cfg[StopTransactionOnInvalidId] = core.ConfigurationKey{Key: StopTransactionOnInvalidId, Readonly: false, Value: strconv.FormatBool(true)}
	cfg[StopTxnAlignedData] = core.ConfigurationKey{Key: StopTxnAlignedData, Readonly: false, Value: strconv.FormatBool(true)}
	cfg[StopTxnSampledData] = core.ConfigurationKey{Key: StopTxnSampledData, Readonly: false, Value: string(types.MeasurandEnergyActiveExportRegister)}
	cfg[SupportedFeatureProfiles] = core.ConfigurationKey{Key: SupportedFeatureProfiles, Readonly: true, Value: fmt.Sprintf("%v,%v,%v,%v,%v,%v", core.ProfileName, firmware.ProfileName, localauth.ProfileName, reservation.ProfileName, remotetrigger.ProfileName, smartcharging.ProfileName)}
	cfg[TransactionMessageAttempts] = core.ConfigurationKey{Key: TransactionMessageAttempts, Readonly: false, Value: strconv.FormatInt(5, intBase)}
	cfg[TransactionMessageRetryInterval] = core.ConfigurationKey{Key: TransactionMessageRetryInterval, Readonly: false, Value: strconv.FormatInt(60, intBase)}
	cfg[UnlockConnectorOnEVSideDisconnect] = core.ConfigurationKey{Key: UnlockConnectorOnEVSideDisconnect, Readonly: false, Value: strconv.FormatBool(true)}
	cfg[WebSocketPingInterval] = core.ConfigurationKey{Key: WebSocketPingInterval, Readonly: false, Value: strconv.FormatInt(54, intBase)}
	cfg[LocalAuthListEnabled] = core.ConfigurationKey{Key: LocalAuthListEnabled, Readonly: false, Value: strconv.FormatBool(true)}
	cfg[LocalAuthListMaxLength] = core.ConfigurationKey{Key: LocalAuthListMaxLength, Readonly: true, Value: strconv.FormatInt(100, intBase)}
	cfg[SendLocalListMaxLength] = core.ConfigurationKey{Key: SendLocalListMaxLength, Readonly: true, Value: strconv.FormatInt(20, intBase)}
	cfg[ChargeProfileMaxStackLevel] = core.ConfigurationKey{Key: ChargeProfileMaxStackLevel, Readonly: true, Value: strconv.FormatInt(10, intBase)}
	cfg[ChargingScheduleAllowedChargingRateUnit] = core.ConfigurationKey{Key: ChargingScheduleAllowedChargingRateUnit, Readonly: true, Value: "Power"}
	cfg[ChargingScheduleMaxPeriods] = core.ConfigurationKey{Key: ChargingScheduleMaxPeriods, Readonly: true, Value: strconv.FormatInt(5, intBase)}
	cfg[MaxChargingProfilesInstalled] = core.ConfigurationKey{Key: MaxChargingProfilesInstalled, Readonly: true, Value: strconv.FormatInt(10, intBase)}
	return cfg
}
