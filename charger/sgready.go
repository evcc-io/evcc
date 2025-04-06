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
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/measurement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// SgReady charger implementation
type SgReady struct {
	*embed
	mode  int64
	modeS func(int64) error
	modeG func() (int64, error)
}

func init() {
	registry.AddCtx("sgready", NewSgReadyFromConfig)
}

const (
	_ int64 = iota
	Normal
	Boost
	Stop
)

//go:generate go tool decorate -f decorateSgReady -b *SgReady -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.SocLimiter,GetLimitSoc,func() (int64, error)"

// NewSgReadyFromConfig creates an SG Ready configurable charger from generic config
func NewSgReadyFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed                   `mapstructure:",squash"`
		SetMode                 plugin.Config
		GetMode                 *plugin.Config // optional
		measurement.Temperature `mapstructure:",squash"`
		measurement.Energy      `mapstructure:",squash"`
		Phases                  int
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

	modeS, err := cc.SetMode.IntSetter(ctx, "mode")
	if err != nil {
		return nil, err
	}

	modeG, err := cc.GetMode.IntGetter(ctx)
	if err != nil {
		return nil, err
	}

	res, err := NewSgReady(ctx, &cc.embed, modeS, modeG)
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
func NewSgReady(ctx context.Context, embed *embed, modeS func(int64) error, modeG func() (int64, error)) (*SgReady, error) {
	res := &SgReady{
		embed: embed,
		mode:  Normal,
		modeS: modeS,
		modeG: modeG,
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

	if mode == Stop {
		return api.StatusNone, errors.New("stop mode")
	}

	status := map[int64]api.ChargeStatus{Boost: api.StatusC, Normal: api.StatusB}
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
	err := wb.modeS(mode)
	if err == nil {
		wb.mode = mode
	}
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *SgReady) MaxCurrent(current int64) error {
	return nil
}
