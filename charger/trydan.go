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

// https://v2charge.com/trydan/

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

type RealTimeData struct {
	ID                 string  `json:"ID"`
	ChargeState        int     `json:"ChargeState"`
	ReadyState         int     `json:"ReadyState"`
	ChargePower        float64 `json:"ChargePower"`
	ChargeEnergy       float64 `json:"ChargeEnergy"`
	SlaveError         int     `json:"SlaveError"`
	ChargeTime         int     `json:"ChargeTime"`
	HousePower         float64 `json:"HousePower"`
	FVPower            float64 `json:"FVPower"`
	BatteryPower       float64 `json:"BatteryPower"`
	Paused             int     `json:"Paused"`
	Locked             int     `json:"Locked"`
	Timer              int     `json:"Timer"`
	Intensity          int     `json:"Intensity"`
	Dynamic            int     `json:"Dynamic"`
	MinIntensity       int     `json:"MinIntensity"`
	MaxIntensity       int     `json:"MaxIntensity"`
	PauseDynamic       int     `json:"PauseDynamic"`
	FirmwareVersion    string  `json:"FirmwareVersion"`
	DynamicPowerMode   int     `json:"DynamicPowerMode"`
	ContractedPower    int     `json:"ContractedPower"`
	ChargeMode         int     `json:"ChargeMode"`
	IntensityMeasureL1 float64 `json:"IntensityMeasure_L1"`
	IntensityMeasureL2 float64 `json:"IntensityMeasure_L2"`
	IntensityMeasureL3 float64 `json:"IntensityMeasure_L3"`
	VoltageMeasureL1   float64 `json:"VoltageMeasure_L1"`
	VoltageMeasureL2   float64 `json:"VoltageMeasure_L2"`
	VoltageMeasureL3   float64 `json:"VoltageMeasure_L3"`
}

// phaseMeasurementsUnavailable reports whether the firmware clearly does not populate
// per-phase current/voltage: power is flowing but every phase current still reads zero,
// which is physically impossible on firmware that actually reports these fields (added
// in 2.5.0; older firmware just omits them, unmarshalling to zero). A zero reading while
// idle is inconclusive either way, so it is not treated as unavailable.
func (data RealTimeData) phaseMeasurementsUnavailable() bool {
	return data.ChargePower > 0 &&
		data.IntensityMeasureL1+data.IntensityMeasureL2+data.IntensityMeasureL3 == 0
}

// Trydan ChargeMode values
const (
	trydanChargeModeMono  = 0
	trydanChargeModeThree = 1
	trydanChargeModeMixed = 2
)

// Trydan charger implementation
type Trydan struct {
	*request.Helper
	uri        string
	statusG    util.Cacheable[RealTimeData]
	current    int
	enabled    bool
	autoUnlock bool
	wasLocked  bool
}

func init() {
	registry.Add("trydan", NewTrydanFromConfig)
}

// NewTrydanFromConfig creates a Trydan charger from generic config
func NewTrydanFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI        string
		Cache      time.Duration
		AutoUnlock bool
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewTrydan(cc.URI, cc.Cache, cc.AutoUnlock)
}

// NewTrydan creates Trydan charger
func NewTrydan(uri string, cache time.Duration, autoUnlock bool) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Trydan{
		Helper:     request.NewHelper(util.NewLogger("trydan")),
		uri:        util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		autoUnlock: autoUnlock,
	}

	c.statusG = util.ResettableCached(func() (RealTimeData, error) {
		var res RealTimeData
		uri := fmt.Sprintf("%s/RealTimeData", c.uri)
		err := c.GetJSON(uri, &res)
		return res, err
	}, cache)

	return c, nil
}

// Status implements the api.Charger interface
func (t Trydan) Status() (api.ChargeStatus, error) {
	data, err := t.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}
	switch state := data.ChargeState; state {
	case 0:
		return api.StatusA, nil
	case 1:
		return api.StatusB, nil
	case 2:
		// firmware keeps ChargeState at "charging" even after Paused=1, so fall
		// back to StatusB while paused to avoid reporting charging when the
		// charger itself confirms no power is flowing (ChargePower drops to 0)
		if data.Paused == 1 {
			return api.StatusB, nil
		}
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown status: %d", state)
	}
}

// Enabled implements the api.Charger interface
func (c Trydan) Enabled() (bool, error) {
	data, err := c.statusG.Get()
	return data.Paused == 0, err
}

func (c *Trydan) setValue(param string, value int) error {
	uri := fmt.Sprintf("%s/write/%s=%d", c.uri, param, value)
	res, err := c.GetBody(uri)
	if str := string(res); err == nil && str != "OK" {
		err = fmt.Errorf("command failed: %s", res)
	}
	return err
}

// Enable implements the api.Charger interface
func (c *Trydan) Enable(enable bool) error {
	var pause, pauseDynamic int
	if !enable {
		pause = 1
	} else {
		pauseDynamic = 1
	}

	data, err := c.statusG.Get()
	if err != nil {
		return err
	}

	// Locked disables the EVSE entirely and is independent of Paused; it must not be
	// coupled to Paused here, or the charger resets its session energy/time counters.
	// It may still be locked by the owner (manually, or via V2C's autolock feature) for
	// security reasons (e.g. public installations). Opt-in via autoUnlock: unlock only
	// when we actually need to start charging, and only re-lock afterwards if we're the
	// ones who unlocked it - this way we never override a lock state the owner set
	// independently of evcc. Left off by default since unlocking releases the physical
	// cable latch on some vehicles (e.g. Tesla), which some setups already handle via
	// their own means (e.g. phone-proximity autolock) and don't want evcc to duplicate.
	if c.autoUnlock {
		switch {
		case enable && data.Locked == 1:
			if err := c.setValue("Locked", 0); err != nil {
				return err
			}
			c.wasLocked = true
		case !enable && c.wasLocked:
			if err := c.setValue("Locked", 1); err != nil {
				return err
			}
			c.wasLocked = false
		}
	}

	if err := c.setValue("Paused", pause); err != nil {
		return err
	}
	// Pause/Unpause Dynamic Power Control if enabled.
	// This is needed to let EVCC taking over charging power control.
	// Charger will stop returning power readings if 'Dynamic' is disabled.
	if data.Dynamic == 1 {
		if err := c.setValue("PauseDynamic", pauseDynamic); err != nil {
			// Pause V2C 'PauseDynamic' when EVCC charging is active and vice versa.
			return err
		}
	}
	c.enabled = enable

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c Trydan) MaxCurrent(current int64) error {
	err := c.setValue("Intensity", int(current))
	if err == nil {
		c.current = int(current)
	}
	return err
}

// removed broken interfaces https://github.com/evcc-io/evcc/issues/28047

// var _ api.ChargeRater = (*Trydan)(nil)

// // ChargedEnergy implements the api.ChargeRater interface
// func (c Trydan) ChargedEnergy() (float64, error) {
// 	data, err := c.statusG.Get()
// 	return data.ChargeEnergy, err
// }

// var _ api.ChargeTimer = (*Trydan)(nil)

// // ChargeDuration implements the api.ChargeTimer interface
// func (c Trydan) ChargeDuration() (time.Duration, error) {
// 	data, err := c.statusG.Get()
// 	return time.Duration(data.ChargeTime) * time.Second, err
// }

var _ api.PhaseSwitcher = (*Trydan)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (c Trydan) Phases1p3p(phases int) error {
	mode := trydanChargeModeThree
	if phases == 1 {
		mode = trydanChargeModeMono
	}
	return c.setValue("ChargeMode", mode)
}

var _ api.PhaseGetter = (*Trydan)(nil)

// GetPhases implements the api.PhaseGetter interface
func (c Trydan) GetPhases() (int, error) {
	data, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	switch data.ChargeMode {
	case trydanChargeModeMono:
		return 1, nil
	case trydanChargeModeThree:
		return 3, nil
	default:
		// mixed mode: phase count varies dynamically, not a fixed 1p/3p state
		return 0, nil
	}
}

var _ api.Meter = (*Trydan)(nil)

// CurrentPower implements the api.Meter interface
func (c Trydan) CurrentPower() (float64, error) {
	data, err := c.statusG.Get()
	return data.ChargePower, err
}

var _ api.PhaseCurrents = (*Trydan)(nil)

// Currents implements the api.PhaseCurrents interface
func (c Trydan) Currents() (float64, float64, float64, error) {
	data, err := c.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	if data.phaseMeasurementsUnavailable() {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return data.IntensityMeasureL1, data.IntensityMeasureL2, data.IntensityMeasureL3, nil
}

var _ api.PhaseVoltages = (*Trydan)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c Trydan) Voltages() (float64, float64, float64, error) {
	data, err := c.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	if data.phaseMeasurementsUnavailable() {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return data.VoltageMeasureL1, data.VoltageMeasureL2, data.VoltageMeasureL3, nil
}

var _ api.Diagnosis = (*Trydan)(nil)

// Diagnose implements the api.Diagnosis interface
func (c *Trydan) Diagnose() {
	data, err := c.statusG.Get()
	if err != nil {
		fmt.Printf("%#v", data)
	}
}
