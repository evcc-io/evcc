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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/measurement"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// SgReady charger implementation
type SgReady struct {
	*embed
	mode  int64
	modeS func(int64) error
	modeG func() (int64, error)

	// optional power setter for devices that support SGReady with power envelope
	power     int64
	lp        loadpoint.API
	maxPowerS func(int64) error
}

func init() {
	registry.AddCtx("sgready", NewSgReadyFromConfig)
}

const (
	_      int64 = iota
	Dim          // 1
	Normal       // 2
	Boost        // 3
)

//go:generate go tool decorate -f decorateSgReady -b *SgReady -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.SocLimiter,GetLimitSoc,func() (int64, error)"

// NewSgReadyFromConfig creates an SG Ready configurable charger from generic config
func NewSgReadyFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed                   `mapstructure:",squash"`
		SetMode                 plugin.Config
		GetMode                 *plugin.Config // optional
		SetMaxPower             *plugin.Config // optional
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

	modeS, err := cc.SetMode.IntSetter(ctx, "mode")
	if err != nil {
		return nil, err
	}

	modeG, err := cc.GetMode.IntGetter(ctx)
	if err != nil {
		return nil, err
	}

	maxPowerS, err := cc.SetMaxPower.IntSetter(ctx, "maxpower")
	if err != nil {
		return nil, err
	}

	res, err := NewSgReady(ctx, &cc.embed, modeS, modeG, maxPowerS)
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

	return decorateSgReady(res, powerG, energyG, tempG, limitTempG), nil
}

// NewSgReady creates SG Ready charger
func NewSgReady(ctx context.Context, embed *embed, modeS func(int64) error, modeG func() (int64, error), maxPowerS func(int64) error) (*SgReady, error) {
	res := &SgReady{
		embed:     embed,
		mode:      Normal,
		modeS:     modeS,
		modeG:     modeG,
		maxPowerS: maxPowerS,
	}

	return res, nil
}

func (wb *SgReady) getMode() (int64, error) {
	if wb.modeG == nil {
		return wb.mode, nil
	}
	return wb.modeG()
}

// Status implements the api.Charger interface
func (wb *SgReady) Status() (api.ChargeStatus, error) {
	mode, err := wb.getMode()
	if err != nil {
		return api.StatusNone, err
	}

	status := map[int64]api.ChargeStatus{
		Dim:    api.StatusB,
		Normal: api.StatusB,
		Boost:  api.StatusC,
	}
	return status[mode], nil
}

// Enabled implements the api.Charger interface
func (wb *SgReady) Enabled() (bool, error) {
	mode, err := wb.getMode()
	return mode == Boost, err
}

// Enable implements the api.Charger interface
func (wb *SgReady) Enable(enable bool) error {
	mode := map[bool]int64{false: Normal, true: Boost}[enable]

	if err := wb.modeS(mode); err != nil {
		return err
	}

	wb.mode = mode

	return wb.setMaxPower(wb.power)
}

var _ api.Dimmer = (*SgReady)(nil)

// Dimmed implements the api.Dimmer interface
func (wb *SgReady) Dimmed() (bool, error) {
	mode, err := wb.getMode()
	return mode == Dim, err
}

// Dimm implements the api.Dimmer interface
func (wb *SgReady) Dim(dim bool) error {
	mode := Normal
	if dim {
		mode = Dim
	}

	if err := wb.modeS(mode); err != nil {
		return err
	}

	wb.mode = Dim

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *SgReady) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SgReady)(nil)

// MaxCurrent implements the api.Charger interface
func (wb *SgReady) MaxCurrentMillis(current float64) error {
	phases := 1
	if wb.lp != nil {
		phases = wb.lp.GetPhases()
	}
	return wb.setMaxPower(int64(230 * current * float64(phases)))
}

func (wb *SgReady) setMaxPower(power int64) error {
	if wb.maxPowerS == nil {
		return nil
	}

	err := wb.maxPowerS(power)
	if err == nil {
		wb.power = power
	}

	return err
}

var _ loadpoint.Controller = (*SgReady)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *SgReady) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
