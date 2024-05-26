package foxesscloud

import "encoding/json"

// Wrappers
type GetDeviceRealTimeData struct {
	SN   string
	Data Variables
	Time string
}

// API
type GetDeviceRealTimeDataResult []struct {
	DeviceSN string `json:"deviceSN"`
	Datas    []Data `json:"datas"`
	Time     string `json:"time"`
}

// Helpers
type GenericResponse struct {
	ErrNo  int32           `json:"errno"`
	Msg    string          `json:"msg"`
	Result json.RawMessage `json:"result"`
}

// https://www.foxesscloud.com/public/i18n/en/OpenApiDocument.html#7
type Variables struct {
	TodayYield *float64 // kW

	PvPower *float64 // kW

	Pv1Volt    *float64 // V
	Pv1Current *float64 // A
	Pv1Power   *float64 // kW

	Pv2Volt    *float64 // V
	Pv2Current *float64 // A
	Pv2Power   *float64 // kW

	Pv3Volt    *float64 // V
	Pv3Current *float64 // A
	Pv3Power   *float64 // kW

	Pv4Volt    *float64 // V
	Pv4Current *float64 // A
	Pv4Power   *float64 // kW

	Pv5Volt    *float64 // V
	Pv5Current *float64 // A
	Pv5Power   *float64 // kW

	Pv6Volt    *float64 // V
	Pv6Current *float64 // A
	Pv6Power   *float64 // kW

	Pv7Volt    *float64 // V
	Pv7Current *float64 // A
	Pv7Power   *float64 // kW

	Pv8Volt    *float64 // V
	Pv8Current *float64 // A
	Pv8Power   *float64 // kW

	Pv9Volt    *float64 // V
	Pv9Current *float64 // A
	Pv9Power   *float64 // kW

	Pv10Volt    *float64 // V
	Pv10Current *float64 // A
	Pv10Power   *float64 // kW

	Pv11Volt    *float64 // V
	Pv11Current *float64 // A
	Pv11Power   *float64 // kW

	Pv12Volt    *float64 // V
	Pv12Current *float64 // A
	Pv12Power   *float64 // kW

	Pv13Volt    *float64 // V
	Pv13Current *float64 // A
	Pv13Power   *float64 // kW

	Pv14Volt    *float64 // V
	Pv14Current *float64 // A
	Pv14Power   *float64 // kW

	Pv15Volt    *float64 // V
	Pv15Current *float64 // A
	Pv15Power   *float64 // kW

	Pv16Volt    *float64 // V
	Pv16Current *float64 // A
	Pv16Power   *float64 // kW

	Pv17Volt    *float64 // V
	Pv17Current *float64 // A
	Pv17Power   *float64 // kW

	Pv18Volt    *float64 // V
	Pv18Current *float64 // A
	Pv18Power   *float64 // kW

	EpsPower *float64 // kW

	EpsCurrentR *float64 // A
	EpsVoltR    *float64 // V
	EpsPowerR   *float64 // kW

	EpsCurrentS *float64 // A
	EpsVoltS    *float64 // V
	EpsPowerS   *float64 // kW

	EpsCurrentT *float64 // A
	EpsVoltT    *float64 // V
	EpsPowerT   *float64 // kW

	RCurrent *float64 // A
	RVolt    *float64 // V
	RFreq    *float64 // Hz
	RPower   *float64 // kW

	SCurrent *float64 // A
	SVolt    *float64 // V
	SFreq    *float64 // Hz
	SPower   *float64 // kW

	TCurrent *float64 // A
	TVolt    *float64 // V
	TFreq    *float64 // Hz
	TPower   *float64 // kW

	AmbientTemperation *float64 // ℃
	BoostTemperation   *float64 // ℃
	InvTemperation     *float64 // ℃
	ChargeTemperature  *float64 // ℃
	BatTemperature     *float64 // ℃
	DspTemperature     *float64 // ℃

	LoadsPower  *float64 // kW
	LoadsPowerR *float64 // kW
	LoadsPowerS *float64 // kW
	LoadsPowerT *float64 // kW

	GenerationPower      *float64 // kW
	FeedinPower          *float64 // kW
	GridConsumptionPower *float64 // kW

	InvBatVolt    *float64 // V
	InvBatCurrent *float64 // A
	InvBatPower   *float64 // kW

	BatChargePower    *float64 // kW
	BatDischargePower *float64 // kW
	BatVolt           *float64 // V
	BatCurrent        *float64 // A

	MeterPower  *float64 // kW
	MeterPower2 *float64 // kW
	MeterPowerR *float64 // kW
	MeterPowerS *float64 // kW
	MeterPowerT *float64 // kW

	SoC            *int     // %
	ReactivePower  *float64 // kVar
	PowerFactor    *int
	Generation     *float64 // kWh
	ResidualEnergy *float64 // 10Wh
	RunningState   string   // Enum: (160-> self-test) (161 -> waiting) (162 -> checking) (163 -> on-grid) (164 -> off-grid) (165 -> fault) (166 -> permanent-fault) (167 -> standby) (168 -> upgrading) (169 -> fct) (170 -> illegal)

	BatStatus   string
	BatStatusV2 string

	CurrentFault      string
	CurrentFaultCount string
}

type Data struct {
	Name     string          `json:"name"`
	Unit     string          `json:"unit"`
	Value    json.RawMessage `json:"value"`
	Variable string          `json:"variable"`
}