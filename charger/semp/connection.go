package semp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util/request"
)

// Connection represents a SEMP HTTP connection with helper methods
type Connection struct {
	helper   *request.Helper
	uri      string
	deviceID string
}

// NewConnection creates a new SEMP client
func NewConnection(helper *request.Helper, uri, deviceID string) *Connection {
	return &Connection{
		helper:   helper,
		uri:      uri,
		deviceID: deviceID,
	}
}

// GetDeviceXML retrieves the complete SEMP document from the base URL
// This is more efficient than making separate requests to /DeviceStatus, /DeviceInfo, /PlanningRequest
func (c *Connection) GetDeviceXML() (Device2EM, error) {
	var response Device2EM
	if err := c.helper.GetXML(c.uri, &response); err != nil {
		return Device2EM{}, err
	}
	return response, nil
}

// GetDeviceStatus retrieves the current device status from SEMP interface
func (c *Connection) GetDeviceStatus() (DeviceStatus, error) {
	response, err := c.GetDeviceXML()
	if err != nil {
		return DeviceStatus{}, err
	}

	// Find device status for our device ID
	for _, status := range response.DeviceStatus {
		if status.DeviceID == c.deviceID {
			return status, nil
		}
	}

	return DeviceStatus{}, fmt.Errorf("device %s not found in status response", c.deviceID)
}

// GetDeviceInfo retrieves the device info from SEMP interface
func (c *Connection) GetDeviceInfo() (DeviceInfo, error) {
	response, err := c.GetDeviceXML()
	if err != nil {
		return DeviceInfo{}, err
	}

	// Find device info for our device ID
	for _, info := range response.DeviceInfo {
		if info.Identification.DeviceID == c.deviceID {
			return info, nil
		}
	}

	return DeviceInfo{}, fmt.Errorf("device %s not found in info response", c.deviceID)
}

// HasPlanningRequest checks if there is a planning request/timeframe for the device
func (c *Connection) HasPlanningRequest() (bool, error) {
	response, err := c.GetDeviceXML()
	if err != nil {
		return false, err
	}

	// Check if there are any timeframes for our device ID
	for _, planningRequest := range response.PlanningRequest {
		for _, timeframe := range planningRequest.Timeframe {
			if timeframe.DeviceID == c.deviceID {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetParametersXML retrieves SEMP parameters from the /Parameters endpoint
func (c *Connection) GetParametersXML() ([]Parameter, error) {
	var response Device2EM
	uri := fmt.Sprintf("%s/Parameters", c.uri)
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
func (c *Connection) SendDeviceControl(on bool, power *int) error {
	control := DeviceControl{
		DeviceID:                    c.deviceID,
		On:                          on,
		RecommendedPowerConsumption: power,
		Timestamp:                   int(time.Now().Unix()),
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
