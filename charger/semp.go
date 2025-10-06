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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/semp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// SEMP charger implementation
type SEMP struct {
	*request.Helper
	log         *util.Logger
	conn        *semp.Connection
	cache       time.Duration
	deviceG     util.Cacheable[semp.Device2EM]
	parametersG util.Cacheable[[]semp.Parameter]
	phases      int
	current     float64
	enabled     bool
	deviceID    string
	minPower    int
	maxPower    int
}

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

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSEMP(cc.URI, cc.DeviceID, cc.Cache)
}

// NewSEMP creates a SEMP charger
func NewSEMP(uri, deviceID string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("semp")

	wb := &SEMP{
		Helper:   request.NewHelper(log),
		log:      log,
		cache:    cache,
		phases:   3,
		enabled:  true,
		deviceID: deviceID,
	}

	// Set default timeout
	wb.Client.Timeout = request.Timeout

	// Initialize SEMP connection
	wb.conn = semp.NewConnection(wb.Helper, strings.TrimRight(uri, "/"), deviceID)

	// Setup cached document getter - fetches the complete SEMP document once
	wb.deviceG = util.ResettableCached(func() (semp.Device2EM, error) {
		return wb.conn.GetDeviceXML()
	}, cache)

	// Setup cached parameters getter
	wb.parametersG = util.ResettableCached(func() ([]semp.Parameter, error) {
		return wb.conn.GetParametersXML()
	}, cache)

	var (
		phases1p3p func(int) error
		getPhases  func() (int, error)
	)

	// Check if device supports phase switching by checking power characteristics
	info, err := wb.getDeviceInfo()
	if err != nil {
		return nil, err
	}

	wb.minPower = info.Characteristics.MinPowerConsumption
	wb.maxPower = info.Characteristics.MaxPowerConsumption

	// Assume Phase switching support if MinPowerConsumption < 4140W and MaxPowerConsumption > 4600W
	if wb.minPower > 0 && wb.minPower < 4140 && wb.maxPower > 4600 {
		phases1p3p = wb.phases1p3p
		getPhases = wb.getPhases
		log.DEBUG.Println("detected phase switching support")
	}

	wb.enabled, err = wb.Enabled()
	if err != nil {
		return nil, err
	}

	return decorateSEMP(wb, phases1p3p, getPhases), nil
}

// getDeviceStatus retrieves device status from cached document
func (wb *SEMP) getDeviceStatus() (semp.DeviceStatus, error) {
	doc, err := wb.deviceG.Get()
	if err != nil {
		return semp.DeviceStatus{}, err
	}

	for _, status := range doc.DeviceStatus {
		if status.DeviceID == wb.deviceID {
			return status, nil
		}
	}

	return semp.DeviceStatus{}, fmt.Errorf("device %s not found in status response", wb.deviceID)
}

// getDeviceInfo retrieves device info from cached document
func (wb *SEMP) getDeviceInfo() (semp.DeviceInfo, error) {
	doc, err := wb.deviceG.Get()
	if err != nil {
		return semp.DeviceInfo{}, err
	}

	for _, info := range doc.DeviceInfo {
		if info.Identification.DeviceID == wb.deviceID {
			return info, nil
		}
	}

	return semp.DeviceInfo{}, fmt.Errorf("device %s not found in info response", wb.deviceID)
}

// hasPlanningRequest checks if planning request exists in cached document
func (wb *SEMP) hasPlanningRequest() (bool, error) {
	doc, err := wb.deviceG.Get()
	if err != nil {
		return false, err
	}

	for _, planningRequest := range doc.PlanningRequest {
		for _, timeframe := range planningRequest.Timeframe {
			if timeframe.DeviceID == wb.deviceID {
				return true, nil
			}
		}
	}

	return false, nil
}

// Status implements the api.Charger interface
func (wb *SEMP) Status() (api.ChargeStatus, error) {
	status, err := wb.getDeviceStatus()
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

	// If status is "On" and power consumption > 0, the charger is actively charging -> Status C
	if status.Status == semp.StatusOn && status.PowerInfo.AveragePower > 0 {
		return api.StatusC, nil
	}

	// Everything else (ready, waiting, etc.) -> Status B
	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *SEMP) Enabled() (bool, error) {
	status, err := wb.getDeviceStatus()
	if err != nil {
		return false, err
	}

	return status.EMSignalsAccepted && status.Status == semp.StatusOn, nil
}

// Enable implements the api.Charger interface
func (wb *SEMP) Enable(enable bool) error {
	// Check if interruptions are allowed first
	info, err := wb.getDeviceInfo()
	if err != nil {
		return err
	}

	status, err := wb.getDeviceStatus()
	if err != nil {
		return err
	}

	if !info.Capabilities.InterruptionsAllowed || !status.EMSignalsAccepted {
		return errors.New("device does not allow control")
	}

	wb.enabled = enable
	err = wb.conn.SendDeviceControl(wb.enabled, wb.calcPower())
	if err == nil {
		wb.deviceG.Reset()
		wb.parametersG.Reset()
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *SEMP) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SEMP)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *SEMP) MaxCurrentMillis(current float64) error {
	wb.current = current
	err := wb.conn.SendDeviceControl(wb.enabled, wb.calcPower())

	wb.deviceG.Reset()
	wb.parametersG.Reset()

	return err
}

var _ api.Meter = (*SEMP)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SEMP) CurrentPower() (float64, error) {
	status, err := wb.getDeviceStatus()
	if err != nil {
		return 0, err
	}

	return status.PowerInfo.AveragePower, nil
}

var _ api.ChargeRater = (*SEMP)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *SEMP) ChargedEnergy() (float64, error) {
	parameters, err := wb.parametersG.Get()
	if err != nil {
		return 0, err
	}

	// Find Measurement.ChaSess.WhIn parameter
	for _, param := range parameters {
		if param.ChannelID == "Measurement.ChaSess.WhIn" {
			var energy float64
			if _, err := fmt.Sscanf(param.Value, "%f", &energy); err != nil {
				return 0, fmt.Errorf("failed to parse energy value '%s': %w", param.Value, err)
			}
			// Convert Wh to kWh
			return energy / 1000, nil
		}
	}

	// Return 0 if parameter not found (device might not support it)
	return 0, api.ErrNotAvailable
}

var _ api.Diagnosis = (*SEMP)(nil)

// Diagnose implements the api.Diagnosis interface
func (s *SEMP) Diagnose() {
	if status, err := s.conn.GetDeviceStatus(); err == nil {
		fmt.Printf("Device Status: %+v\n", status)
	} else {
		fmt.Printf("Device Status Error: %v\n", err)
	}

	if info, err := s.conn.GetDeviceInfo(); err == nil {
		fmt.Printf("Device Info: %+v\n", info)
	} else {
		fmt.Printf("Device Info Error: %v\n", err)
	}

	if hasPlanning, err := s.conn.HasPlanningRequest(); err == nil {
		fmt.Printf("Planning Request: %t\n", hasPlanning)
	} else {
		fmt.Printf("Planning Request Error: %v\n", err)
	}
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *SEMP) phases1p3p(phases int) error {
	// SEMP protocol doesn't have explicit phase switching
	wb.phases = phases
	err := wb.conn.SendDeviceControl(wb.enabled, wb.calcPower())

	wb.deviceG.Reset()
	wb.parametersG.Reset()

	return err
}

func (wb *SEMP) getPhases() (int, error) {
	return wb.phases, nil
}

func (wb *SEMP) calcPower() int {
	if !wb.enabled {
		return 0
	}

	return min(max(int(230*float64(wb.phases)*wb.current), wb.minPower), wb.maxPower)
}
