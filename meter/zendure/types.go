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

type Command struct {
	CommandTopic      string `json:"command_topic"`
	DeviceClass       string `json:"device_class"`
	ElectricLevel     int    `json:"electricLevel"`
	Name              string `json:"name"`
	PayloadOff        bool   `json:"payload_off"`
	PayloadOn         bool   `json:"payload_on"`
	Sn                string `json:"sn"`
	StateOff          bool   `json:"state_off"`
	StateOn           bool   `json:"state_on"`
	StateTopic        string `json:"state_topic"`
	UniqueId          string `json:"unique_id"`
	UnitOfMeasurement string `json:"unit_of_measurement"`
	ValueTemplate     string `json:"value_template"`
}
