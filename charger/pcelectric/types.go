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
	AccSessionMillis       int    //  0,
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
		SessionStartValue      int    //  -1,
		AccSessionMillis       int    //  0,
		CurrentChargingCurrent int    //  -1,
		CurrentChargingPower   int    //  0,
		NrOfPhases             int    //  1,
		TwinSerial             int    //  -1,
	}
	TwinCharger interface{}
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

// {
//   "success": 0,
//   "accEnergy": 169100,
//   "phase1Current": 158,
//   "phase2Current": 157,
//   "phase3Current": 156,
//   "phase1InstPower": 3,
//   "phase2InstPower": 3,
//   "phase3InstPower": 3,
//   "readTime": 1640783279023,
//   "gridNetType": "UNKNOWN",
//   "meterSerial": "116223V",
//   "type": 341,
//   "apparentPower": 9
// }
