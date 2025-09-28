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
	log        *util.Logger
	sempClient *semp.Client
	cache      time.Duration
	statusG    util.Cacheable[semp.DeviceStatus]
	infoG      util.Cacheable[semp.DeviceInfo]
	phases     int
	current    float64
	enabled    bool
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
		Helper: request.NewHelper(log),
		log:    log,
		cache:  cache,
		phases: 3,
	}

	// Set default timeout
	wb.Client.Timeout = request.Timeout

	// Initialize SEMP client
	wb.sempClient = semp.NewClient(wb.Helper, strings.TrimRight(uri, "/"), deviceID)

	// Setup cached device status getter
	wb.statusG = util.ResettableCached(func() (semp.DeviceStatus, error) {
		return wb.sempClient.GetDeviceStatus()
	}, cache)

	// Setup cached device info getter
	wb.infoG = util.ResettableCached(func() (semp.DeviceInfo, error) {
		return wb.sempClient.GetDeviceInfo()
	}, cache)

	var (
		phases1p3p func(int) error
		getPhases  func() (int, error)
	)

	// Check if device supports phase switching by checking power characteristics
	info, err := wb.sempClient.GetDeviceInfo()
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

// Status implements the api.Charger interface
func (wb *SEMP) Status() (api.ChargeStatus, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	// Check if there is a planning request/timeframe for this device
	// If no planning request exists -> Status A (unplugged/disconnected)
	hasPlanningRequest, err := wb.sempClient.HasPlanningRequest()
	if err != nil {
		return api.StatusNone, err
	}

	if !hasPlanningRequest {
		return api.StatusA, nil
	}

	// If status is "On", the charger is actively charging -> Status C
	if status.Status == semp.StatusOn {
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

	return status.EMSignalsAccepted && status.Status == semp.StatusOn, nil
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
	err = wb.sempClient.SendDeviceControl(wb.enabled, wb.calcPower())
	if err == nil {
		wb.statusG.Reset()
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
	err := wb.sempClient.SendDeviceControl(wb.enabled, wb.calcPower())
	if err == nil {
		wb.statusG.Reset()
	}
	return err
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
	if status, err := s.sempClient.GetDeviceStatus(); err == nil {
		fmt.Printf("Device Status: %+v\n", status)
	} else {
		fmt.Printf("Device Status Error: %v\n", err)
	}

	if info, err := s.sempClient.GetDeviceInfo(); err == nil {
		fmt.Printf("Device Info: %+v\n", info)
	} else {
		fmt.Printf("Device Info Error: %v\n", err)
	}

	if hasPlanning, err := s.sempClient.HasPlanningRequest(); err == nil {
		fmt.Printf("Planning Request: %t\n", hasPlanning)
	} else {
		fmt.Printf("Planning Request Error: %v\n", err)
	}
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *SEMP) phases1p3p(phases int) error {
	// SEMP protocol doesn't have explicit phase switching
	wb.phases = phases
	err := wb.sempClient.SendDeviceControl(wb.enabled, wb.calcPower())
	if err == nil {
		wb.statusG.Reset()
	}
	return err
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
