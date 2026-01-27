package ecoflow

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util/request"
)

const certificationURL = "/iot-open/sign/certification"

// MQTTCredentials from EcoFlow certification API
type MQTTCredentials struct {
	Account  string `json:"certificateAccount"`
	Password string `json:"certificatePassword"`
	URL      string `json:"url"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
}

// GetMQTTCredentials fetches MQTT credentials from EcoFlow certification API
func GetMQTTCredentials(helper *request.Helper, uri string) (*MQTTCredentials, error) {
	uri = strings.TrimSuffix(uri, "/")
	url := fmt.Sprintf("%s%s", uri, certificationURL)

	var res response[MQTTCredentials]
	if err := helper.GetJSON(url, &res); err != nil {
		return nil, fmt.Errorf("mqtt certification: %w", err)
	}

	if res.Code != "0" {
		return nil, fmt.Errorf("mqtt certification failed: %s - %s", res.Code, res.Message)
	}

	return &res.Data, nil
}

// BrokerURL returns the full MQTT broker URL
func (c *MQTTCredentials) BrokerURL() string {
	protocol := c.Protocol
	if protocol == "mqtts" {
		protocol = "tls"
	}
	return fmt.Sprintf("%s://%s:%s", protocol, c.URL, c.Port)
}
