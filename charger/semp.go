package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// SEMP charger implementation
type SEMP struct {
	*request.Helper
	log      *util.Logger
	uri      string
	deviceID string
	cache    time.Duration
	statusG  util.Cacheable[DeviceStatus]
	infoG    util.Cacheable[DeviceInfo]
	phases   int
	current  float64
	enabled  bool
}

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

// Status constants
const (
	StatusOn  = "On"
	StatusOff = "Off"
)

//go:generate go tool decorate -f decorateSEMP -b *SEMP -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

func init() {
	registry.Add("semp", NewSEMPFromConfig)
}

// NewSEMPFromConfig creates a SEMP charger from generic config
func NewSEMPFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI      string
		DeviceID string
		Cache    time.Duration
	}{
		Cache: 5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	if cc.DeviceID == "" {
		return nil, errors.New("missing deviceId")
	}

	return NewSEMP(cc.URI, cc.DeviceID, cc.Cache)
}

// NewSEMP creates a SEMP charger
func NewSEMP(uri, deviceID string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("semp")

	wb := &SEMP{
		Helper:   request.NewHelper(log),
		log:      log,
		uri:      strings.TrimRight(uri, "/"),
		deviceID: deviceID,
		cache:    cache,
	}

	// Set default timeout
	wb.Client.Timeout = request.Timeout

	// Setup cached device status getter
	wb.statusG = util.ResettableCached(func() (DeviceStatus, error) {
		return wb.getDeviceStatus()
	}, cache)

	// Setup cached device info getter
	wb.infoG = util.ResettableCached(func() (DeviceInfo, error) {
		return wb.getDeviceInfo()
	}, cache)

	var (
		phases1p3p func(int) error
		getPhases  func() (int, error)
	)

	// Check if device supports phase switching by checking power characteristics
	info, err := wb.getDeviceInfo()
	if err == nil {
		// Assume Phase switching support if MinPowerConsumption < 4140W and MaxPowerConsumption > 4600W
		if info.Characteristics.MinPowerConsumption > 0 && info.Characteristics.MinPowerConsumption < 4140 &&
			info.Characteristics.MaxPowerConsumption > 4600 {
			phases1p3p = wb.phases1p3p
			getPhases = wb.getPhases
			log.DEBUG.Println("detected phase switching support")
		}
	}

	return decorateSEMP(wb, phases1p3p, getPhases), nil
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

// DoXML executes HTTP request and decodes XML response
func (wb *SEMP) DoXML(req *http.Request, res interface{}) error {
	resp, err := wb.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := request.ResponseError(resp); err != nil {
		return err
	}

	return xml.NewDecoder(resp.Body).Decode(&res)
}

// getDeviceStatus retrieves the current device status from SEMP interface
func (wb *SEMP) getDeviceStatus() (DeviceStatus, error) {
	uri := fmt.Sprintf("%s/semp/DeviceStatus", wb.uri)

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptXML)
	if err != nil {
		return DeviceStatus{}, err
	}

	var response Device2EM
	if err := wb.DoXML(req, &response); err != nil {
		return DeviceStatus{}, err
	}

	// Find device status for our device ID
	for _, status := range response.DeviceStatus {
		if status.DeviceID == wb.deviceID {
			return status, nil
		}
	}

	return DeviceStatus{}, fmt.Errorf("device %s not found in status response", wb.deviceID)
}

// getDeviceInfo retrieves the device info from SEMP interface
func (wb *SEMP) getDeviceInfo() (DeviceInfo, error) {
	uri := fmt.Sprintf("%s/semp/DeviceInfo", wb.uri)

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptXML)
	if err != nil {
		return DeviceInfo{}, err
	}

	var response Device2EM
	if err := wb.DoXML(req, &response); err != nil {
		return DeviceInfo{}, err
	}

	// Find device info for our device ID
	for _, info := range response.DeviceInfo {
		if info.Identification.DeviceID == wb.deviceID {
			return info, nil
		}
	}

	return DeviceInfo{}, fmt.Errorf("device %s not found in info response", wb.deviceID)
}

// hasPlanningRequest checks if there is a planning request/timeframe for the device
func (wb *SEMP) hasPlanningRequest() (bool, error) {
	uri := fmt.Sprintf("%s/semp/PlanningRequest", wb.uri)

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptXML)
	if err != nil {
		return false, err
	}

	var response Device2EM
	if err := wb.DoXML(req, &response); err != nil {
		return false, err
	}

	// Check if there are any timeframes for our device ID
	for _, planningRequest := range response.PlanningRequest {
		for _, timeframe := range planningRequest.Timeframe {
			if timeframe.DeviceID == wb.deviceID {
				return true, nil
			}
		}
	}

	return false, nil
}

// sendDeviceControl sends a control message to the SEMP device
func (wb *SEMP) sendDeviceControl(on bool, power int) error {
	control := DeviceControl{
		DeviceID:                    wb.deviceID,
		On:                          on,
		RecommendedPowerConsumption: power,
		Timestamp:                   int(time.Now().Unix()),
	}

	message := EM2Device{
		Xmlns:         "http://www.sma.de/communication/schema/SEMP/v1",
		DeviceControl: []DeviceControl{control},
	}

	uri := fmt.Sprintf("%s/semp/DeviceControl", wb.uri)

	req, err := request.New(http.MethodPost, uri, MarshalXML(message), request.XMLEncoding)
	if err != nil {
		return err
	}

	_, err = wb.DoBody(req)
	if err == nil {
		wb.statusG.Reset()
	}

	return err
}

// Status implements the api.Charger interface
func (wb *SEMP) Status() (api.ChargeStatus, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	// Check if there is a planning request/timeframe for this device
	// If no planning request exists -> Status A (unplugged/disconnected)
	hasPlanningRequest, err := wb.hasPlanningRequest()
	if err != nil {
		return api.StatusNone, err
	}

	if !hasPlanningRequest {
		return api.StatusA, nil
	}

	// If status is "On", the charger is actively charging -> Status C
	if status.Status == StatusOn {
		return api.StatusC, nil
	}

	// Everything else (ready, waiting, etc.) -> Status B
	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *SEMP) Enabled() (bool, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return false, err
	}

	return status.EMSignalsAccepted && status.Status == StatusOn, nil
}

// Enable implements the api.Charger interface
func (wb *SEMP) Enable(enable bool) error {
	// Check if interruptions are allowed first
	info, err := wb.infoG.Get()
	if err != nil {
		return err
	}

	status, err := wb.statusG.Get()
	if err != nil {
		return err
	}

	if !info.Capabilities.InterruptionsAllowed || !status.EMSignalsAccepted {
		return errors.New("device does not allow control")
	}

	wb.enabled = enable
	return wb.sendDeviceControl(wb.enabled, wb.calcPower())
}

// MaxCurrent implements the api.Charger interface
func (wb *SEMP) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SEMP)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *SEMP) MaxCurrentMillis(current float64) error {
	wb.current = current
	return wb.sendDeviceControl(wb.enabled, wb.calcPower())
}

var _ api.Meter = (*SEMP)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SEMP) CurrentPower() (float64, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return 0, err
	}

	return float64(status.PowerInfo.AveragePower), nil
}

var _ api.Diagnosis = (*SEMP)(nil)

// Diagnose implements the api.Diagnosis interface
func (s *SEMP) Diagnose() {
	if status, err := s.getDeviceStatus(); err == nil {
		fmt.Printf("Device Status: %+v\n", status)
	}

	if info, err := s.getDeviceInfo(); err == nil {
		fmt.Printf("Device Info: %+v\n", info)
	}

	if hasPlanning, err := s.hasPlanningRequest(); err == nil {
		fmt.Printf("Planning Request: %t\n", hasPlanning)
	}
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *SEMP) phases1p3p(phases int) error {
	// SEMP protocol doesn't have explicit phase switching
	wb.phases = phases
	return wb.sendDeviceControl(wb.enabled, wb.calcPower())
}

func (wb *SEMP) getPhases() (int, error) {
	return wb.phases, nil
}

func (wb *SEMP) calcPower() int {
	if !wb.enabled {
		return 0
	}

	return int(230 * float64(wb.phases) * wb.current)
}
