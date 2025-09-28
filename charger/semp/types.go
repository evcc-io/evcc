package semp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util/request"
)

// Status constants
const (
	StatusOn  = "On"
	StatusOff = "Off"
)

// DeviceInfo represents SEMP device information
type DeviceInfo struct {
	Identification  Identification  `xml:"Identification"`
	Characteristics Characteristics `xml:"Characteristics"`
	Capabilities    Capabilities    `xml:"Capabilities"`
}

// Identification represents SEMP device identification
type Identification struct {
	DeviceID     string `xml:"DeviceId"`
	DeviceName   string `xml:"DeviceName"`
	DeviceType   string `xml:"DeviceType"`
	DeviceSerial string `xml:"DeviceSerial"`
	DeviceVendor string `xml:"DeviceVendor"`
}

// Characteristics represents SEMP device characteristics
type Characteristics struct {
	MinPowerConsumption int `xml:"MinPowerConsumption"`
	MaxPowerConsumption int `xml:"MaxPowerConsumption"`
	MinOnTime           int `xml:"MinOnTime,omitempty"`
	MinOffTime          int `xml:"MinOffTime,omitempty"`
}

// Capabilities represents SEMP device capabilities
type Capabilities struct {
	CurrentPowerMethod   string `xml:"CurrentPower>Method"`
	AbsoluteTimestamps   bool   `xml:"Timestamps>AbsoluteTimestamps"`
	InterruptionsAllowed bool   `xml:"Interruptions>InterruptionsAllowed"`
	OptionalEnergy       bool   `xml:"Requests>OptionalEnergy"`
}

// DeviceStatus represents SEMP device status
type DeviceStatus struct {
	DeviceID          string    `xml:"DeviceId"`
	Status            string    `xml:"Status"`
	EMSignalsAccepted bool      `xml:"EMSignalsAccepted"`
	PowerInfo         PowerInfo `xml:"PowerConsumption>PowerInfo"`
}

// PowerInfo represents SEMP power information
type PowerInfo struct {
	AveragePower      int `xml:"AveragePower"`
	Timestamp         int `xml:"Timestamp"`
	AveragingInterval int `xml:"AveragingInterval"`
}

// DeviceControl represents SEMP device control message
type DeviceControl struct {
	DeviceID                    string `xml:"DeviceId"`
	On                          bool   `xml:"On"`
	RecommendedPowerConsumption int    `xml:"RecommendedPowerConsumption"`
	Timestamp                   int    `xml:"Timestamp"`
}

// PlanningRequest represents SEMP planning request
type PlanningRequest struct {
	Timeframe []Timeframe `xml:"Timeframe"`
}

// Timeframe represents SEMP timeframe
type Timeframe struct {
	DeviceID            string `xml:"DeviceId"`
	EarliestStart       int    `xml:"EarliestStart"`
	LatestEnd           int    `xml:"LatestEnd"`
	MinRunningTime      *int   `xml:"MinRunningTime,omitempty"`
	MaxRunningTime      *int   `xml:"MaxRunningTime,omitempty"`
	MinEnergy           *int   `xml:"MinEnergy,omitempty"`
	MaxEnergy           *int   `xml:"MaxEnergy,omitempty"`
	MaxPowerConsumption *int   `xml:"MaxPowerConsumption,omitempty"`
	MinPowerConsumption *int   `xml:"MinPowerConsumption,omitempty"`
}

// Device2EM represents the device to energy manager message
type Device2EM struct {
	XMLName         xml.Name          `xml:"Device2EM"`
	Xmlns           string            `xml:"xmlns,attr"`
	DeviceInfo      []DeviceInfo      `xml:"DeviceInfo,omitempty"`
	DeviceStatus    []DeviceStatus    `xml:"DeviceStatus,omitempty"`
	PlanningRequest []PlanningRequest `xml:"PlanningRequest,omitempty"`
}

// EM2Device represents the energy manager to device message
type EM2Device struct {
	XMLName       xml.Name        `xml:"EM2Device"`
	Xmlns         string          `xml:"xmlns,attr"`
	DeviceControl []DeviceControl `xml:"DeviceControl,omitempty"`
}

// MarshalXML marshals XML into an io.ReadSeeker
func MarshalXML(data interface{}) io.ReadSeeker {
	if data == nil {
		return nil
	}

	body, err := xml.Marshal(data)
	if err != nil {
		return &errorReader{err: err}
	}

	return bytes.NewReader(body)
}

// errorReader wraps an error with an io.Reader
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func (r *errorReader) Seek(offset int64, whence int) (int64, error) {
	return 0, r.err
}

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

// DoXML executes HTTP request and decodes XML response
func (c *Client) DoXML(req *http.Request, res interface{}) error {
	resp, err := c.helper.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := request.ResponseError(resp); err != nil {
		return err
	}

	return xml.NewDecoder(resp.Body).Decode(&res)
}

// GetDeviceStatus retrieves the current device status from SEMP interface
func (c *Client) GetDeviceStatus() (DeviceStatus, error) {
	uri := fmt.Sprintf("%s/semp/DeviceStatus", c.uri)

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptXML)
	if err != nil {
		return DeviceStatus{}, err
	}

	var response Device2EM
	if err := c.DoXML(req, &response); err != nil {
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

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptXML)
	if err != nil {
		return DeviceInfo{}, err
	}

	var response Device2EM
	if err := c.DoXML(req, &response); err != nil {
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

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptXML)
	if err != nil {
		return false, err
	}

	var response Device2EM
	if err := c.DoXML(req, &response); err != nil {
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

	req, err := request.New(http.MethodPost, uri, MarshalXML(message), request.XMLEncoding)
	if err != nil {
		return err
	}

	_, err = c.helper.DoBody(req)
	return err
}
