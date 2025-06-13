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
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/measurement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

// SgReadyBoost charger implementation
type SgReadyBoost struct {
	*embed
}

func init() {
	registry.AddCtx("sgready-boost", NewSgReadyBoostFromConfig)
}

// NewSgReadyBoostFromConfig creates an SG Ready charger with boost relay from generic config
func NewSgReadyBoostFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed                   `mapstructure:",squash"`
		Config                  config.Typed
		measurement.Temperature `mapstructure:",squash"`
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	charger, err := NewFromConfig(ctx, cc.Config.Type, cc.Config.Other)
	if err != nil {
		return nil, err
	}

	tempG, limitTempG, err := cc.Temperature.Configure(ctx)
	if err != nil {
		return nil, err
	}

	res, err := NewSgReadyBoost(ctx, &cc.embed, charger)
	if err != nil {
		return nil, err
	}

	return decorateSgReady(res, nil, nil, tempG, limitTempG), nil
}

// NewSgReadyBoost creates SG Ready charger with boost relay
func NewSgReadyBoost(ctx context.Context, embed *embed, charger api.Charger) (*SgReady, error) {
	modeS := func(mode int64) error {
		switch mode {
		case Dimm:
			return api.ErrNotAvailable
		case Normal:
			return charger.Enable(false)
		case Boost:
			return charger.Enable(true)
		default:
			return fmt.Errorf("invalid sgready mode: %d", mode)
		}
	}

	modeG := func() (int64, error) {
		enabled, err := charger.Enabled()
		if err != nil {
			return 0, err
		}
		if enabled {
			return Boost, nil
		}
		return Normal, nil
	}

	return NewSgReady(ctx, embed, modeS, modeG, nil)
}
