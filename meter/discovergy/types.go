package discovergy

const API = "https://api.inexogy.com/public/v1"

type Meter struct {
	MeterID          string `json:"meterId"`
	SerialNumber     string `json:"serialNumber"`
	FullSerialNumber string `json:"fullSerialNumber"`
}

type Reading struct {
	Time   int64
	Values struct {
		EnergyOut                    int64
		Energy1, Energy2             int64
		Voltage1, Voltage2, Voltage3 int64
		EnergyOut1, EnergyOut2       int64
		Power1, Power2, Power3       int64
		Power                        int64
		Energy                       int64
	}
}
