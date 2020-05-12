package nrgble

const (
	InfoService           = "8f75bba0-c903-11e4-9fe8-0002a5d6b15d"
	EnergyService         = "0379e580-ad1b-11e4-8bdd-0002a5d6b15d"
	PowerService          = "fd005380-b065-11e4-9ce2-0002a5d6b15d"
	VoltageCurrentService = "171bad00-b066-11e4-aeda-0002a5d6b15d"
	SettingsService       = "14b3afc0-ad1b-11e4-baab-0002a5d6b15d"
)

type Info struct {
	Current              int  `struc:"int8"`  // appInfo[0]
	KWhPer100            int  `struc:"int16"` // appInfo[1] / 10
	AmountPerKWh         int  `struc:"int8"`  // appInfo[2] / 100
	FIEnabled            int  `struc:"int8"`  // 1 == appInfo[3]
	ErrorCode            int  `struc:"int8"`  // appInfo[4]
	Efficiency           int  `struc:"int8"`  // appInfo[5]
	ChargingActive       bool `struc:"int8"`  // 1 == appInfo[6]
	PauseCharging        bool `struc:"int8"`  // 1 == appInfo[7]
	ChargingCurrentMax   int  `struc:"int8"`  // appInfo[8]
	BLETransmissionPower int  `struc:"int8"`  // appInfo[9]
}

type Energy struct {
	TotalEnergy         int `struc:"uint32"` // energie02[0] / 1000
	EnergyLastCharge    int `struc:"uint32"` // energie02[1] / 1000
	Energy2ndLastCharge int `struc:"uint32"` // energie02[2] / 1000
	Energy3rdLastCharge int `struc:"uint32"` // energie02[3] / 1000
	ChargingEnergyLimit int `struc:"uint16"` // energie02[4] / 100
}

type Power struct {
	TotalPower        int `struc:"uint16"` // leistung[0] / 100
	L1                int `struc:"uint16"` // leistung[1] / 100
	L2                int `struc:"uint16"` // leistung[2] / 100
	L3                int `struc:"uint16"` // leistung[3] / 100
	PeakPower         int `struc:"uint16"` // leistung[4] / 100;
	Frequency         int `struc:"uint16"` // leistung[5] / 100
	Temperature       int `struc:"int16"`  // leistung[6]
	RemainingDistance int `struc:"uint16"` // leistung[7] / 10
	Costs             int `struc:"uint16"` // leistung[8] / 100
	CPSignal          int `struc:"int8"`   // round(((leistung[9] << 8) / 100) + 0.4, 1)
}

type VoltageCurrent struct {
	VoltageL1 int `struc:"uint16"` // voltageAndCurrent[0] / 10
	VoltageL2 int `struc:"uint16"` // voltageAndCurrent[1] / 10
	VoltageL3 int `struc:"uint16"` // voltageAndCurrent[2] / 10
	CurrentL1 int `struc:"uint16"` // voltageAndCurrent[3] / 100
	CurrentL2 int `struc:"uint16"` // voltageAndCurrent[4] / 100
	CurrentL3 int `struc:"uint16"` // voltageAndCurrent[5] / 100
}

type Settings struct {
	PIN                  int  `struc:"uint16"` // H SettingsPIN
	Current              int  `struc:"uint8"`  // B SettingsChargingCurrentValue
	ChargingEnergyLimit  int  `struc:"uint16"` // H SettingsChargingEnergyOff
	KWhPer100            int  `struc:"uint16"` // H round(SettingsKWhPer100Value * 10)
	AmountPerKWh         int  `struc:"uint8"`  // B round(SettingsAmountPerKWhValue * 100)
	Pad                  int  `struc:"[2]pad"` // xx
	Efficiency           int  `struc:"uint8"`  // B SettingsEfficacyValue
	PauseCharging        bool `struc:"uint8"`  // B 1 if SettingsPauseCharging else 0
	BLETransmissionPower int  `struc:"int8"`   // b SettingsBLETransmissionPowerValue
	PadTail              int  `struc:"[5]pad"` // xxxxx
}
