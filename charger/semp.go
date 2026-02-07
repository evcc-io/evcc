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
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/semp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

// SEMP charger implementation
type SEMP struct {
	log            *util.Logger
	conn           *semp.Connection
	cache          time.Duration
	deviceG        util.Cacheable[semp.Device2EM]
	parametersG    util.Cacheable[[]semp.Parameter]
	phases         int
	current        float64
	enabled        bool
	deviceID       string
	minPower       int
	maxPower       int
	hasStatusParam bool
}

//go:generate go tool decorate -f decorateSEMP -b *SEMP -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)"

func init() {
	registry.AddCtx("semp", NewSEMPFromConfig)
}

// NewSEMPFromConfig creates a SEMP charger from generic config
func NewSEMPFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
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

	return NewSEMP(ctx, cc.URI, cc.DeviceID, cc.Cache)
}

// NewSEMP creates a SEMP charger
func NewSEMP(ctx context.Context, uri, deviceID string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("semp")

	wb := &SEMP{
		log:      log,
		cache:    cache,
		phases:   3,
		current:  6,
		enabled:  true,
		deviceID: deviceID,
	}

	// Initialize SEMP connection
	wb.conn = semp.NewConnection(log, uri)

	// Setup cached device getter
	wb.deviceG = util.ResettableCached(func() (semp.Device2EM, error) {
		return wb.conn.GetDeviceXML()
	}, cache)

	// Setup cached parameters getter
	wb.parametersG = util.ResettableCached(func() ([]semp.Parameter, error) {
		return wb.conn.GetParametersXML()
	}, cache)

	// Auto-detect deviceID if not configured
	if deviceID == "" {
		doc, err := wb.deviceG.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve device info: %w", err)
		}

		if len(doc.DeviceInfo) == 0 {
			return nil, fmt.Errorf("no device info found")
		}

		// Use first device ID found
		wb.deviceID = doc.DeviceInfo[0].Identification.DeviceID
		log.DEBUG.Printf("auto-detected device ID: %s", wb.deviceID)
	}

	var (
		phases1p3p    func(int) error
		getPhases     func() (int, error)
		chargedEnergy func() (float64, error)
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
	}

	// Check if device supports charged energy reporting via Parameters endpoint
	if _, err := wb.getParameter("Measurement.ChaSess.WhIn"); err == nil {
		chargedEnergy = wb.chargedEnergy
	}

	// Check if device supports status reporting via Parameters endpoint
	if _, err := wb.getParameter("Measurement.Operation.EVeh.ChaStt"); err == nil {
		wb.hasStatusParam = true
	}

	wb.enabled, err = wb.Enabled()
	if err != nil {
		return nil, err
	}

	go wb.heartbeat(ctx)

	return decorateSEMP(wb, phases1p3p, getPhases, chargedEnergy), nil
}

// heartbeat ensures that device control updates are sent at least once per minute
func (wb *SEMP) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if we need to send an update
			if time.Since(wb.conn.Updated()) >= time.Minute {
				if err := wb.conn.SendDeviceControl(wb.deviceID, wb.calcPower(wb.enabled, wb.current, wb.phases)); err != nil {
					wb.log.ERROR.Printf("heartbeat: failed to send update: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
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

// getParameter retrieves a specific parameter value by channel ID
func (wb *SEMP) getParameter(channelID string) (string, error) {
	parameters, err := wb.parametersG.Get()
	if err != nil {
		return "", err
	}

	for _, param := range parameters {
		if param.ChannelID == channelID {
			return param.Value, nil
		}
	}

	return "", fmt.Errorf("parameter %s not found", channelID)
}

func (wb *SEMP) calcPower(enabled bool, current float64, phases int) int {
	if !enabled || current == 0 {
		return 0
	}

	return min(max(int(230*float64(phases)*current), wb.minPower), wb.maxPower)
}

// Status implements the api.Charger interface
func (wb *SEMP) Status() (api.ChargeStatus, error) {
	if wb.hasStatusParam {
		value, err := wb.getParameter("Measurement.Operation.EVeh.ChaStt")
		if err != nil {
			return api.StatusNone, err
		}

		switch value {
		case "200111": // "nicht verbunden"
			return api.StatusA, nil
		case "200112": // "verbunden"
			return api.StatusB, nil
		case "200113": // "Ladevorgang aktiv"
			return api.StatusC, nil
		default:
			return api.StatusNone, fmt.Errorf("unknown status value: %s", value)
		}
	}

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

	if status.EMSignalsAccepted {
		if status.Status == semp.StatusOn {
			return true, nil
		}

		// work around wrong status reporting during phase switching (SMA Chargers...)
		return wb.enabled, nil
	}

	return false, nil
}

// Enable implements the api.Charger interface
func (wb *SEMP) Enable(enable bool) error {
	if err := wb.conn.SendDeviceControl(wb.deviceID, wb.calcPower(enable, wb.current, wb.phases)); err != nil {
		return err
	}

	wb.enabled = enable
	wb.deviceG.Reset()
	wb.parametersG.Reset()

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *SEMP) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SEMP)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *SEMP) MaxCurrentMillis(current float64) error {
	if err := wb.conn.SendDeviceControl(wb.deviceID, wb.calcPower(wb.enabled, current, wb.phases)); err != nil {
		return err
	}

	wb.current = current
	wb.deviceG.Reset()
	wb.parametersG.Reset()

	return nil
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

// chargedEnergy implements the api.ChargeRater interface (via decorator)
func (wb *SEMP) chargedEnergy() (float64, error) {
	value, err := wb.getParameter("Measurement.ChaSess.WhIn")
	if err != nil {
		return 0, err
	}

	var energy float64
	if _, err := fmt.Sscanf(value, "%f", &energy); err != nil {
		return 0, fmt.Errorf("failed to parse energy value '%s': %w", value, err)
	}

	// Convert Wh to kWh
	return energy / 1000, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *SEMP) phases1p3p(phases int) error {
	// SEMP protocol doesn't have explicit phase switching
	if err := wb.conn.SendDeviceControl(wb.deviceID, wb.calcPower(wb.enabled, wb.current, phases)); err != nil {
		return err
	}

	wb.phases = phases
	wb.deviceG.Reset()
	wb.parametersG.Reset()

	return nil
}

func (wb *SEMP) getPhases() (int, error) {
	return wb.phases, nil
}

var _ api.Diagnosis = (*SEMP)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SEMP) Diagnose() {
	if status, err := wb.getDeviceStatus(); err == nil {
		fmt.Printf("Device Status: %+v\n", status)
	} else {
		fmt.Printf("Device Status Error: %v\n", err)
	}

	if info, err := wb.getDeviceInfo(); err == nil {
		fmt.Printf("Device Info: %+v\n", info)
	} else {
		fmt.Printf("Device Info Error: %v\n", err)
	}

	if hasPlanning, err := wb.hasPlanningRequest(); err == nil {
		fmt.Printf("Planning Request: %t\n", hasPlanning)
	} else {
		fmt.Printf("Planning Request Error: %v\n", err)
	}
}
