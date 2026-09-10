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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/lgthinq"
	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
)

// LgThinq controls an LG ThinQ water heater by raising its target temperature during boost
type LgThinq struct {
	*SgReady
}

func init() {
	registry.AddCtx("lg-thinq", NewLgThinqFromConfig)
}

// NewLgThinqFromConfig creates an LG ThinQ water heater charger from generic config
func NewLgThinqFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		Token    string
		Country  string
		Device   string
		Setpoint float64
		Cache    time.Duration
	}{
		embed: embed{
			Icon_:     "waterheater",
			Features_: []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice, api.SwitchDevice},
		},
		Country:  "DE",
		Setpoint: 60,
		Cache:    time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Token == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("lg-thinq").Redact(cc.Token)
	conn := lgthinq.NewAPI(log, cc.Token, cc.Country)

	device, err := ensureEx("device", cc.Device, func() ([]lgthinq.Device, error) {
		devices, err := conn.Devices()
		return lo.Filter(devices, func(d lgthinq.Device, _ int) bool {
			return d.DeviceInfo.DeviceType == lgthinq.DeviceTypeWaterHeater
		}), err
	}, func(d lgthinq.Device) (string, error) {
		return d.DeviceId, nil
	})
	if err != nil {
		return nil, err
	}

	deviceId := device.DeviceId

	stateG := util.ResettableCached(func() (lgthinq.WaterHeaterState, error) {
		var res lgthinq.WaterHeaterState
		err := conn.State(deviceId, &res)
		return res, err
	}, cc.Cache)

	// target temperature before boost, restored when boost ends
	var normal float64

	set := func(mode int64) error {
		switch mode {
		case Normal:
			if normal == 0 {
				return nil
			}

			if err := conn.WaterHeaterTargetTemperature(deviceId, normal); err != nil {
				return err
			}

			normal = 0
			stateG.Reset()
			return nil

		case Boost:
			state, err := stateG.Get()
			if err != nil {
				return err
			}

			if temp, ok := state.Celsius(); ok && normal == 0 {
				normal = temp.TargetTemperature
			}

			if err := conn.WaterHeaterTargetTemperature(deviceId, cc.Setpoint); err != nil {
				return err
			}

			stateG.Reset()
			return nil

		default:
			return api.ErrNotAvailable
		}
	}

	res := new(LgThinq)

	res.SgReady, err = NewSgReady(ctx, &cc.embed, set, nil, nil)
	if err != nil {
		return nil, err
	}

	implement.Has(res, implement.Battery(func() (float64, error) {
		state, err := stateG.Get()
		if err != nil {
			return 0, err
		}

		temp, ok := state.Celsius()
		if !ok {
			return 0, api.ErrNotAvailable
		}

		return temp.CurrentTemperature, nil
	}))

	return res, nil
}
