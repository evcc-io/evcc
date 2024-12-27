package zendure

type CredentialsRequest struct {
	SnNumber string `json:"snNumber"`
	Account  string `json:"account"`
}

type CredentialsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AppKey  string `json:"appKey"`
		Secret  string `json:"secret"`
		MqttUrl string `json:"mqttUrl"`
		Port    int    `json:"port"`
	}
	Msg string `json:"msg"`
}

type Payload struct {
	*Command
	*Data
}

type Command struct {
	CommandTopic      string `json:"command_topic"`
	DeviceClass       string `json:"device_class"`
	Name              string `json:"name"`
	PayloadOff        bool   `json:"payload_off"`
	PayloadOn         bool   `json:"payload_on"`
	StateOff          bool   `json:"state_off"`
	StateOn           bool   `json:"state_on"`
	StateTopic        string `json:"state_topic"`
	UniqueId          string `json:"unique_id"`
	UnitOfMeasurement string `json:"unit_of_measurement"`
	ValueTemplate     string `json:"value_template"`
}

type Data struct {
	AcMode          int    `json:"acMode"`          // 1,
	BuzzerSwitch    bool   `json:"buzzerSwitch"`    // false,
	ElectricLevel   int    `json:"electricLevel"`   // 7,
	GridInputPower  int    `json:"gridInputPower"`  // 99,
	HeatState       int    `json:"heatState"`       // 0,
	HubState        int    `json:"hubState"`        // 0,
	HyperTmp        int    `json:"hyperTmp"`        // 2981,
	InputLimit      int    `json:"inputLimit"`      // 100,
	InverseMaxPower int    `json:"inverseMaxPower"` // 1200,
	MasterSwitch    bool   `json:"masterSwitch"`    // true,
	OutputLimit     int    `json:"outputLimit"`     // 0,
	OutputPackPower int    `json:"outputPackPower"` // 70,
	PackInputPower  int    `json:"packInputPower"`  // 70,
	OutputHomePower int    `json:"outputHomePower"` // 70,
	PackNum         int    `json:"packNum"`         // 1,
	PackState       int    `json:"packState"`       // 0,
	RemainInputTime int    `json:"remainInputTime"` // 59940,
	RemainOutTime   int    `json:"remainOutTime"`   // 59940,
	Sn              string `json:"sn"`              // "EE1LH",
	SocSet          int    `json:"socSet"`          // 1000,
	SolarInputPower int    `json:"solarInputPower"` // 0,
	WifiState       bool   `json:"wifiState"`       // true
}
