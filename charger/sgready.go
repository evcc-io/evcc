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
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// SgReady charger implementation
type SgReady struct {
	*embed
	_mode    int64
	phases   int
	get      func() (int64, error)
	set      func(int64) error
	maxPower func(int64) error
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
		embed     `mapstructure:",squash"`
		SetMode   plugin.Config
		GetMode   *plugin.Config // optional
		MaxPower  *plugin.Config // optional
		Power     *plugin.Config // optional
		Energy    *plugin.Config // optional
		Temp      *plugin.Config // optional
		LimitTemp *plugin.Config // optional
		Phases    int
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

	set, err := cc.SetMode.IntSetter(ctx, "mode")
	if err != nil {
		return nil, err
	}

	get, err := cc.GetMode.IntGetter(ctx)
	if err != nil {
		return nil, err
	}

	maxPower, err := cc.MaxPower.IntSetter(ctx, "maxpower")
	if err != nil {
		return nil, err
	}

	res, err := NewSgReady(ctx, &cc.embed, set, get, maxPower, cc.Phases)
	if err != nil {
		return nil, err
	}

	// decorate power
	powerG, err := cc.Power.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("power: %w", err)
	}

	// decorate energy
	energyG, err := cc.Energy.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("energy: %w", err)
	}

	// decorate temp
	tempG, err := cc.Temp.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("temp: %w", err)
	}

	limitTempG, err := cc.LimitTemp.IntGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("limit temp: %w", err)
	}

	return decorateSgReady(res, powerG, energyG, tempG, limitTempG), nil
}

// NewSgReady creates SG Ready charger
func NewSgReady(ctx context.Context, embed *embed, set func(int64) error, get func() (int64, error), maxPower func(int64) error, phases int) (*SgReady, error) {
	res := &SgReady{
		embed:    embed,
		_mode:    Normal,
		set:      set,
		get:      get,
		maxPower: maxPower,
		phases:   phases,
	}

	return res, nil
}

func (wb *SgReady) mode() (int64, error) {
	if wb.get == nil {
		return wb._mode, nil
	}
	return wb.get()
}

// Status implements the api.Charger interface
func (wb *SgReady) Status() (api.ChargeStatus, error) {
	mode, err := wb.mode()
	if err != nil {
		return api.StatusNone, err
	}

	if mode == Stop {
		return api.StatusNone, errors.New("stop mode")
	}

	status := map[int64]api.ChargeStatus{Boost: api.StatusC, Normal: api.StatusB}[mode]
	return status, nil
}

// Enabled implements the api.Charger interface
func (wb *SgReady) Enabled() (bool, error) {
	mode, err := wb.mode()
	return mode == Boost, err
}

// Enable implements the api.Charger interface
func (wb *SgReady) Enable(enable bool) error {
	mode := map[bool]int64{false: Normal, true: Boost}[enable]
	err := wb.set(mode)
	if err == nil {
		wb._mode = mode
	}
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *SgReady) MaxCurrent(current int64) error {
	return wb.MaxCurrentEx(float64(current))
}

// MaxCurrent implements the api.Charger interface
func (wb *SgReady) MaxCurrentEx(current float64) error {
	if wb.maxPower == nil {
		return nil
	}

	return wb.maxPower(int64(230 * current * float64(wb.phases)))
}
