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
