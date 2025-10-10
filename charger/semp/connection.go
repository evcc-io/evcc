package semp

import (
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util/request"
)

// Connection represents a SEMP HTTP connection with helper methods
type Connection struct {
	helper *request.Helper
	uri    string
}

// NewConnection creates a new SEMP client
func NewConnection(helper *request.Helper, uri string) *Connection {
	// Ensure URI ends with exactly one trailing slash
	uri = strings.TrimRight(uri, "/") + "/"

	return &Connection{
		helper: helper,
		uri:    uri,
	}
}

// GetDeviceXML retrieves the complete SEMP document from the base URL
func (c *Connection) GetDeviceXML() (Device2EM, error) {
	var response Device2EM
	if err := c.helper.GetXML(c.uri, &response); err != nil {
		return Device2EM{}, err
	}
	return response, nil
}

// GetParametersXML retrieves device parameters from the /Parameters endpoint
func (c *Connection) GetParametersXML() ([]Parameter, error) {
	var response Device2EM
	uri := c.uri + "Parameters"
	if err := c.helper.GetXML(uri, &response); err != nil {
		return nil, err
	}

	if response.Parameters == nil {
		return []Parameter{}, nil
	}

	return response.Parameters.Parameter, nil
}

// SendDeviceControl sends a control message to the SEMP device
// power is optional - if nil, RecommendedPowerConsumption will be omitted
func (c *Connection) SendDeviceControl(deviceId string, power int) error {
	control := DeviceControl{
		DeviceID:  deviceId,
		On:        power > 0,
		Timestamp: int(time.Now().Unix()),
	}

	if power > 0 {
		control.RecommendedPowerConsumption = &power
	}

	message := EM2Device{
		Xmlns:         "http://www.sma.de/communication/schema/SEMP/v1",
		DeviceControl: []DeviceControl{control},
	}

	req, err := request.New(http.MethodPost, c.uri, request.MarshalXML(message), request.XMLEncoding)
	if err != nil {
		return err
	}

	_, err = c.helper.DoBody(req)
	return err
}
