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
	"github.com/evcc-io/evcc/charger/heating"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// Heatpump charger implementation
type Heatpump struct {
	*embed
	*heating.PowerModeController
	*heating.PowerController
}

func init() {
	registry.AddCtx("heatpump", NewHeatpumpFromConfig)
}

//go:generate go tool decorate -f decorateHeatpump -b *Heatpump -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.SocLimiter,GetLimitSoc,func() (int64, error)"

// NewHeatpumpFromConfig creates an SG Ready configurable charger from generic config
func NewHeatpumpFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed            `mapstructure:",squash"`
		SetMaxPower      plugin.Config
		GetMaxPower      *plugin.Config // optional
		heating.Readings `mapstructure:",squash"`
		Phases           int
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
		},
		Phases: 1,
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

	res, err := NewHeatpump(ctx, &cc.embed, maxPowerS, maxPowerG, cc.Phases)
	if err != nil {
		return nil, err
	}

	powerG, energyG, tempG, limitTempG, err := cc.Readings.Configure(ctx)
	if err != nil {
		return nil, err
	}

	return decorateHeatpump(res, powerG, energyG, tempG, limitTempG), nil
}

// NewHeatpump creates heatpump charger
func NewHeatpump(ctx context.Context, embed *embed, setMaxPower func(int64) error, getMaxPower func() (int64, error), phases int) (*Heatpump, error) {
	pc := heating.NewPowerController(ctx, setMaxPower, phases)

	res := &Heatpump{
		embed:               embed,
		PowerModeController: heating.NewPowerModeController(ctx, pc, getMaxPower),
		PowerController:     pc,
	}

	return res, nil
}
