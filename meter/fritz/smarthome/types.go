package smarthome

// Device represents a smarthome device from the /devices endpoint
type Device struct {
	UID             string   `json:"UID"`
	AIN             string   `json:"ain"`
	Name            string   `json:"name"`
	ProductName     string   `json:"productName"`
	ProductCategory string   `json:"productCategory"`
	IsConnected     bool     `json:"isConnected"`
	UnitUids        []string `json:"unitUids"`
}

// Unit represents a smarthome unit with its interfaces
type Unit struct {
	GroupUID    string      `json:"groupUid,omitempty"`
	UID         string      `json:"UID,omitempty"`
	DeviceUID   string      `json:"deviceUid"`
	UnitType    string      `json:"unitType"`
	IsConnected bool        `json:"isConnected"`
	Statistics  *Statistics `json:"statistics,omitempty"`
	Interfaces  *Interfaces `json:"interfaces,omitempty"`
}

type Interfaces struct {
	MultimeterInterface  *MultimeterInterface  `json:"multimeterInterface,omitempty"`
	OnOffInterface       *OnOffInterface       `json:"onOffInterface,omitempty"`
	TemperatureInterface *TemperatureInterface `json:"temperatureInterface,omitempty"`
}

type Statistics struct {
	Temperatures []ElementFloat `json:"temperatures,omitempty"`
	Powers       []Element      `json:"powers,omitempty"`
	Voltages     []Element      `json:"voltages,omitempty"`
	Energies     []Element      `json:"energies,omitempty"`
}

type ElementFloat struct {
	Interval      int64     `json:"interval"`
	StasticsState string    `json:"statisticsState"`
	Period        string    `json:"period"`
	Values        []float64 `json:"values,omitempty"`
}

type Element struct {
	Interval      int64   `json:"interval"`
	StasticsState string  `json:"statisticsState"`
	Period        string  `json:"period"`
	Values        []int64 `json:"values,omitempty"`
}

// MultimeterInterface contains power/energy measurements
type MultimeterInterface struct {
	State   string  `json:"state"`
	Power   float64 `json:"power"`   // W
	Voltage float64 `json:"voltage"` // V
	Current float64 `json:"current"` // A
	Energy  float64 `json:"energy"`  // Wh
}

// OnOffInterface contains switch state
type OnOffInterface struct {
	State string `json:"state"` // "on" or "off"
}

type TemperatureInterface struct {
	State   string  `json:"state"` // "on" or "off"
	Celsius float64 `json:"celsius"`
}
