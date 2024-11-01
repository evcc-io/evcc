package zendure

type CredentialsRequest struct {
	SnNumber string `json:"snNumber"`
	Account  string `json:"account"`
}

type CredentialsResponse struct {
	Success string `json:"success"` // true,
	Data    struct {
		AppKey  string `json:"appKey"`  // "zendure",
		Secret  string `json:"secret"`  // "zendureSecret",
		MqttUrl string `json:"mqttUrl"` // "mqtt.zen-iot.com",
		Port    int    `json:"port"`    // 1883
	}
	Msg string `json:"msg"` // "Successful operation"
}
