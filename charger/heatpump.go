package charger

// LICENSE

// Copyright (c) 2024 andig

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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/measurement"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// Heatpump charger implementation
type Heatpump struct {
	*embed
	lp        loadpoint.API
	power     int64
	maxPowerG func() (int64, error)
	maxPowerS func(int64) error
}

func init() {
	registry.AddCtx("heatpump", NewHeatpumpFromConfig)
}

//go:generate go tool decorate -f decorateHeatpump -b *Heatpump -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.SocLimiter,GetLimitSoc,func() (int64, error)"

// NewHeatpumpFromConfig creates heatpump configurable charger from generic config
func NewHeatpumpFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed                   `mapstructure:",squash"`
		SetMaxPower             plugin.Config
		GetMaxPower             *plugin.Config // optional
		measurement.Temperature `mapstructure:",squash"`
		measurement.Energy      `mapstructure:",squash"`
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	maxPowerG, err := cc.GetMaxPower.IntGetter(ctx)
	if err != nil {
		return nil, err
	}

	maxPowerS, err := cc.SetMaxPower.IntSetter(ctx, "maxpower")
	if err != nil {
		return nil, err
	}

	// if !sponsor.IsAuthorized() {
	// 	return nil, api.ErrSponsorRequired
	// }

	res, err := NewHeatpump(ctx, &cc.embed, maxPowerS, maxPowerG)
	if err != nil {
		return nil, err
	}

	powerG, energyG, err := cc.Energy.Configure(ctx)
	if err != nil {
		return nil, err
	}

	tempG, limitTempG, err := cc.Temperature.Configure(ctx)
	if err != nil {
		return nil, err
	}

	return decorateHeatpump(res, powerG, energyG, tempG, limitTempG), nil
}

// NewHeatpump creates heatpump charger
func NewHeatpump(ctx context.Context, embed *embed, maxPowerS func(int64) error, maxPowerG func() (int64, error)) (*Heatpump, error) {
	res := &Heatpump{
		embed:     embed,
		maxPowerG: maxPowerG,
		maxPowerS: maxPowerS,
	}

	return res, nil
}

func (wb *Heatpump) getMaxPower() (int64, error) {
	if wb.maxPowerG == nil {
		return wb.power, nil
	}
	return wb.maxPowerG()
}

func (wb *Heatpump) setMaxPower(power int64) error {
	err := wb.maxPowerS(power)
	if err == nil {
		wb.power = power
	}

	return err
}

// Status implements the api.Charger interface
func (wb *Heatpump) Status() (api.ChargeStatus, error) {
	power, err := wb.getMaxPower()
	if err != nil {
		return api.StatusNone, err
	}

	status := map[bool]api.ChargeStatus{false: api.StatusB, true: api.StatusC}
	return status[power > 0], nil
}

// Enabled implements the api.Charger interface
func (wb *Heatpump) Enabled() (bool, error) {
	power, err := wb.getMaxPower()
	return power > 0, err
}

// Enable implements the api.Charger interface
func (wb *Heatpump) Enable(enable bool) error {
	var power int64
	if enable {
		power = wb.power
	}
	return wb.setMaxPower(power)
}

// MaxCurrent implements the api.Charger interface
func (wb *Heatpump) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Heatpump)(nil)

// MaxCurrent implements the api.Charger interface
func (wb *Heatpump) MaxCurrentMillis(current float64) error {
	phases := 1
	if wb.lp != nil {
		phases = wb.lp.GetPhases()
	}
	return wb.setMaxPower(int64(230 * current * float64(phases)))
}

var _ loadpoint.Controller = (*Heatpump)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *Heatpump) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
