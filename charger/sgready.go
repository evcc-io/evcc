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
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// SgReady charger implementation
type SgReady struct {
	get func() (int64, error)
	set func(int64) error
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

//go:generate go run ../cmd/tools/decorate.go -f decorateSgReady -b api.Charger -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.SocLimiter,GetLimitSoc,func() (int64, error)"

// NewSgReadyFromConfig creates an SG Ready configurable charger from generic config
func NewSgReadyFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		GetMode   provider.Config
		SetMode   provider.Config
		Power     *provider.Config // optional
		Energy    *provider.Config // optional
		Temp      *provider.Config // optional
		LimitTemp *provider.Config // optional
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	res, err := NewSgReady(ctx, cc.GetMode, cc.SetMode)
	if err != nil {
		return nil, err
	}

	// decorate power
	var powerG func() (float64, error)
	if cc.Power != nil {
		powerG, err = provider.NewFloatGetterFromConfig(ctx, *cc.Power)
		if err != nil {
			return nil, fmt.Errorf("power: %w", err)
		}
	}

	// decorate energy
	var energyG func() (float64, error)
	if cc.Energy != nil {
		energyG, err = provider.NewFloatGetterFromConfig(ctx, *cc.Energy)
		if err != nil {
			return nil, fmt.Errorf("energy: %w", err)
		}
	}

	// decorate temp
	var tempG func() (float64, error)
	if cc.Temp != nil {
		tempG, err = provider.NewFloatGetterFromConfig(ctx, *cc.Temp)
		if err != nil {
			return nil, fmt.Errorf("temp: %w", err)
		}
	}

	var limitTempG func() (int64, error)
	if cc.LimitTemp != nil {
		limitTempG, err = provider.NewIntGetterFromConfig(ctx, *cc.LimitTemp)
		if err != nil {
			return nil, fmt.Errorf("limit temp: %w", err)
		}
	}

	return decorateSgReady(res, powerG, energyG, tempG, limitTempG), nil
}

// NewSgReady creates SG Ready charger
func NewSgReady(ctx context.Context, get, set provider.Config) (api.Charger, error) {
	getMode, err := provider.NewIntGetterFromConfig(ctx, get)
	if err != nil {
		return nil, err
	}

	setMode, err := provider.NewIntSetterFromConfig(ctx, "mode", set)
	if err != nil {
		return nil, err
	}

	return &SgReady{
		get: getMode,
		set: setMode,
	}, nil
}

// Status implements the api.Charger interface
func (wb *SgReady) Status() (api.ChargeStatus, error) {
	mode, err := wb.get()
	if err != nil {
		return api.StatusNone, err
	}

	if mode == Stop {
		return api.StatusE, errors.New("stop mode")
	}

	status := map[int64]api.ChargeStatus{Boost: api.StatusC, Normal: api.StatusB}[mode]
	return status, nil
}

// Enabled implements the api.Charger interface
func (wb *SgReady) Enabled() (bool, error) {
	mode, err := wb.get()
	return mode == Boost, err
}

// Enable implements the api.Charger interface
func (wb *SgReady) Enable(enable bool) error {
	mode := map[bool]int64{true: Normal, false: Boost}[enable]
	return wb.set(mode)
}

// MaxCurrent implements the api.Charger interface
func (wb *SgReady) MaxCurrent(current int64) error {
	return nil
}
