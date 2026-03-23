package pcelectric

// /servlet/rest/chargebox/status

type Status struct {
	SerialNumber           int64  //  2216247,
	Connector              string //  "NOT_CONNECTED",
	Mode                   string //  "ALWAYS_ON",
	CurrentLimit           int    //  11,
	FactoryCurrentLimit    int    //  32,
	SwitchCurrentLimit     int    //  32,
	PowerMode              string //  "OFF",
	CurrentChargingCurrent int    //  -1,
	CurrentChargingPower   int    //  -1,
	AccSessionEnergy       int    //  0,
	SessionStartTime       int64  //  1641489661842
	ChargeboxTime          string //  "22:25"
	AccSessionMillis       int64  //  0,
	LatestReading          int    //  0,
	ChargeStatus           int    //  0,
	NrOfPhases             int    //  1,
	MainCharger            struct {
		Reference              string //  "Garage",
		SerialNumber           int    //  2216247,
		LastContact            int64  //  1640781615305,
		Online                 bool   //  false,
		LoadBalanced           bool   //  true,
		Phase                  int    //  16,
		ProductId              int    //  121,
		MeterStatus            int    //  1,
		MeterSerial            string //  "",
		ChargeStatus           int    //  0,
		PilotLevel             int    //  6,
		AccEnergy              int    //  -1,
		Connector              string //  "NOT_CONNECTED",
		AccSessionEnergy       int    //  0,
		SessionStartValue      int64  //  -1,
		AccSessionMillis       int64  //  0,
		CurrentChargingCurrent int    //  -1,
		CurrentChargingPower   int    //  0,
		NrOfPhases             int    //  1,
		TwinSerial             int    //  -1,
	}
	TwinCharger any
}

// /servlet/rest/chargebox/slaves/false
type SlaveStatus []struct {
	Reference              string //  "Garage",
	SerialNumber           int    //  2216247,
	LastContact            int64  //  1640781615305,
	Online                 bool   //  false,
	LoadBalanced           bool   //  true,
	Phase                  int    //  16,
	ProductId              int    //  121,
	MeterStatus            int    //  1,
	MeterSerial            string //  "",
	ChargeStatus           int    //  0,
	PilotLevel             int    //  6,
	AccEnergy              int    //  -1,
	FirmwareVersion        int    //  7
	FirmwareRevision       int    //  9
	WifiCardStatus         int    //  2
	Connector              string //  "NOT_CONNECTED",
	AccSessionEnergy       int    //  0,
	SessionStartValue      int    //  -1,
	AccSessionMillis       int64  //  0,
	SessionStartTime       int64  //  1641136092090
	CurrentChargingCurrent int    //  -1,
	CurrentChargingPower   int    //  0,
	NrOfPhases             int    //  1,
	TwinSerial             int    //  -1,
	CableLockMode          int    //  0
	MinCurrentLimit        int    //  6
	DipSwitchSettings      int    //  8188
}

type ReducedIntervals struct {
	ReducedIntervalsEnabled bool                     `json:"reducedIntervalsEnabled"`
	ReducedCurrentIntervals []ReducedCurrentInterval `json:"reducedCurrentIntervals,omitempty"`
}

type ReducedCurrentInterval struct {
	SchemaId    int    `json:"schemaId"`
	Start       string `json:"start"`
	Stop        string `json:"stop"`
	Weekday     int    `json:"weekday"`
	ChargeLimit int    `json:"chargeLimit"`
}

// /servlet/rest/chargebox/meterinfo/CENTRAL100

type MeterInfo struct {
	Success         int    // 0,
	AccEnergy       int64  // 169100,
	Phase1Current   int    // 158,
	Phase2Current   int    // 157,
	Phase3Current   int    // 156,
	Phase1InstPower int    // 3,
	Phase2InstPower int    // 3,
	Phase3InstPower int    // 3,
	ReadTime        int64  // 1640783279023,
	GridNetType     string // "UNKNOWN",
	MeterSerial     string // "116223V",
	Type            int    // 341,
	ApparentPower   int    // 9
}

// /servlet/rest/chargebox/lbconfig/false
type LbConfigShort struct {
	LoadBalancingFuse     int  `json:"loadBalancingFuse"`     // 32
	LoadBalancingPower    int  `json:"loadBalancingPower"`    // 0
	LoadBalancingFuse101  int  `json:"loadBalancingFuse101"`  // 32
	LoadBalancingPower101 int  `json:"loadBalancingPower101"` // 0
	MasterPhase           int  `json:"masterPhase"`           // 16
	MasterLoadBalanced    bool `json:"masterLoadBalanced"`    // true
}

type LbConfig struct {
	LoadBalancingFuse     int  // 16
	LoadBalancingPower    int  // 0
	LoadBalancingFuse101  int  // 16
	LoadBalancingPower101 int  // 0
	MasterPhase           int  // 16
	MasterLoadBalanced    bool // true
	Slaves                []struct {
		Reference              string // "Garage"
		SerialNumber           int    // 2216247
		LastContact            int64  // 1640970181816
		Online                 bool   // false
		LoadBalanced           bool   // true
		Phase                  int    // 16
		ProductId              int    // 121
		MeterStatus            int    //	1
		MeterSerial            string // ""
		ChargeStatus           int    // 144
		PilotLevel             int    // 6
		AccEnergy              int    // -1
		FirmwareVersion        int    // 7
		FirmwareRevision       int    // 9
		WifiCardStatus         int    // 2
		Connector              string // "UNAVAILABLE"
		AccSessionEnergy       int    // 0
		SessionStartValue      int    // -1
		AccSessionMillis       int64  // 174723
		SessionStartTime       int64  // 1640957660208
		CurrentChargingCurrent int    // -1
		CurrentChargingPower   int    // 0
		NrOfPhases             int    // 1
		TwinSerial             int    // -1
		CableLockMode          int    // 0
		MinCurrentLimit        int    // 6
		DipSwitchSettings      int    // 8188
	}
}
type Config struct {
	OcppConnected           bool   // false
	MaxChargeCurrent        int    // 32
	ProductId               string // "81"
	ProgramVersion          string // "7.9"
	FirmwareVersion         int    // 7
	FirmwareRevision        int    // 9
	SmallFirmwareVersion    int    // 2
	SmallFirmwareRevision   int    // 15
	LargeFirmwareVersion    int    // 7
	LargeFirmwareRevision   int    // 9
	LbVersion2              bool   // true
	SerialNumber            int    // 2216247
	MeterSerialNumber       string // ""
	MeterType               int    // 0
	FactoryChargeLimit      int    // 32
	SwitchChargeLimit       int    // 32
	RfidReaderPresent       bool   // true
	RfidMode                string // "RFID_WIFI"
	PowerbackupStatus       int    // 2
	LastTemperature         int    // 25
	WarningTemperature      int    // 65
	CutoffTemperature       int    // 70
	ReducedIntervalsEnabled bool   // true
	ReducedCurrentIntervals []struct {
		SchemaId    int    // 1
		Start       string // "00:00:00"
		Stop        string // "00:00:00"
		Weekday     int    // 8
		ChargeLimit int    // 6
	}
	SoftwareVersion      int    // 185
	AvailableVersion     int    // 185
	UpdateUrl            string // "http%3A%2F%2F85.11.39.104%2Fchargebox_185.tgz"
	NetworkMode          int    // 1
	NetworkType          int    // 0
	NetworkSSID          string // "SchlockeNet"
	WebNetworkPassword   string // ""
	NetworkAPChannel     int    // 6
	EthNetworkMode       int    // 0
	GcConfigTimestamp    int64  // null
	GcloudActivated      bool   // false
	GcActivatedFrom      int64  // null
	EthGateway           string // ""
	EthDNS               string // ""
	EthIP                string // ""
	EthMask              int    // 24
	LocalLoadBalanced    bool   // false
	GroupLoadBalanced    bool   // true
	GroupLoadBalanced101 bool   // false
	EnergyReportEnabled  bool   // false
	Master               bool   // true
	Timezone             string // null
	SlaveList            []struct {
		Reference              string // Garage
		SerialNumber           int    // 2216247
		LastContact            int64  // 1640960502344
		Online                 bool   // false
		LoadBalanced           bool   // true
		Phase                  int    // 16
		ProductId              int    // 121
		MeterStatus            int    // 1
		MeterSerial            string // ""
		ChargeStatus           int    // 144
		PilotLevel             int    // 6
		AccEnergy              int    // -1
		FirmwareVersion        int    // 7
		FirmwareRevision       int    // 9
		WifiCardStatus         int    // 2
		Connector              string // "UNAVAILABLE"
		AccSessionEnergy       int    //	0
		SessionStartValue      int    // -1
		AccSessionMillis       int    // 5635
		SessionStartTime       int64  // 1640957660208
		CurrentChargingCurrent int    // -1
		CurrentChargingPower   int    // 0
		NrOfPhases             int    // 1
		TwinSerial             int    // -1
		CableLockMode          int    // 0
		MinCurrentLimit        int    // 6
		DipSwitchSettings      int    // 8188
	}
}

type MinCurrentLimitStruct []struct {
	MinCurrentLimit int `json:"minCurrentLimit"`
	SerialNumber    int `json:"serialNumber"`
	TwinSerial      int `json:"twinSerial"` // -1
}
