package semp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util/request"
)

// Client represents a SEMP HTTP client with helper methods
type Client struct {
	helper   *request.Helper
	uri      string
	deviceID string
}

// NewClient creates a new SEMP client
func NewClient(helper *request.Helper, uri, deviceID string) *Client {
	return &Client{
		helper:   helper,
		uri:      uri,
		deviceID: deviceID,
	}
}

// GetDeviceStatus retrieves the current device status from SEMP interface
func (c *Client) GetDeviceStatus() (DeviceStatus, error) {
	uri := fmt.Sprintf("%s/semp/DeviceStatus", c.uri)

	var response Device2EM
	if err := c.helper.GetXML(uri, &response); err != nil {
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
func (c *Client) GetDeviceInfo() (DeviceInfo, error) {
	uri := fmt.Sprintf("%s/semp/DeviceInfo", c.uri)

	var response Device2EM
	if err := c.helper.GetXML(uri, &response); err != nil {
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
func (c *Client) HasPlanningRequest() (bool, error) {
	uri := fmt.Sprintf("%s/semp/PlanningRequest", c.uri)

	var response Device2EM
	if err := c.helper.GetXML(uri, &response); err != nil {
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

// SendDeviceControl sends a control message to the SEMP device
func (c *Client) SendDeviceControl(on bool, power int) error {
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

	uri := fmt.Sprintf("%s/semp/DeviceControl", c.uri)

	req, err := request.New(http.MethodPost, uri, request.MarshalXML(message), request.XMLEncoding)
	if err != nil {
		return err
	}

	_, err = c.helper.DoBody(req)
	return err
}
