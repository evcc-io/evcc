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
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

// HeidelBridge charger implementation via the HeidelBridge MQTT API.
// See https://github.com/BorisBrock/HeidelBridge
type HeidelBridge struct {
	implement.Caps
	log      *util.Logger
	statusG  func() (string, error)
	enableS  func(string) error
	currentS func(float64) error
	enabled  bool
}

func init() {
	registry.AddCtx("heidelbridge", NewHeidelBridgeFromConfig)
}

// NewHeidelBridgeFromConfig creates a HeidelBridge charger from generic config
func NewHeidelBridgeFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
	}{
		Topic:   "HeidelBridge",
		Timeout: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHeidelBridge(ctx, cc.Config, cc.Topic, cc.Timeout)
}

// NewHeidelBridge creates a HeidelBridge charger
func NewHeidelBridge(ctx context.Context, mqttconf mqtt.Config, topic string, timeout time.Duration) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("heidel")

	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	wb := &HeidelBridge{
		Caps: implement.New(),
		log:  log,
	}

	mq := func(s string, args ...any) *plugin.Mqtt {
		return plugin.NewMqtt(log, client, fmt.Sprintf(s, args...), timeout).WithContext(ctx)
	}

	// read topics
	wb.statusG, err = mq("%s/vehicle_state", topic).StringGetter()
	if err != nil {
		return nil, err
	}

	// write topics (commands are not retained)
	wb.enableS, err = plugin.NewMqtt(log, client, fmt.Sprintf("%s/control/enable_charging", topic), 0).
		WithContext(ctx).StringSetter("enable")
	if err != nil {
		return nil, err
	}

	wb.currentS, err = plugin.NewMqtt(log, client, fmt.Sprintf("%s/control/charging_current_limit", topic), 0).
		WithContext(ctx).FloatSetter("current")
	if err != nil {
		return nil, err
	}

	// meter (always available)
	powerG, err := mq("%s/charging_power", topic).FloatGetter()
	if err != nil {
		return nil, err
	}
	implement.May(wb, implement.Meter(powerG))

	energyG, err := mq("%s/energy_meter", topic).FloatGetter()
	if err != nil {
		return nil, err
	}
	implement.May(wb, implement.MeterEnergy(energyG))

	// phase currents and voltages
	currents, err := phaseGetters(mq, topic, "charging_current")
	if err != nil {
		return nil, err
	}
	implement.May(wb, implement.PhaseCurrents(currents))

	voltages, err := phaseGetters(mq, topic, "charging_voltage")
	if err != nil {
		return nil, err
	}
	implement.May(wb, implement.PhaseVoltages(voltages))

	return wb, nil
}

// phaseGetters builds a per-phase getter for the given HeidelBridge measurement group
func phaseGetters(mq func(string, ...any) *plugin.Mqtt, topic, group string) (func() (float64, float64, float64, error), error) {
	var g [3]func() (float64, error)
	for i := range g {
		var err error
		if g[i], err = mq("%s/%s/phase%d", topic, group, i+1).FloatGetter(); err != nil {
			return nil, err
		}
	}

	return func() (float64, float64, float64, error) {
		var res [3]float64
		for i, getter := range g {
			var err error
			if res[i], err = getter(); err != nil {
				return 0, 0, 0, err
			}
		}
		return res[0], res[1], res[2], nil
	}, nil
}

// Status implements the api.Charger interface
func (wb *HeidelBridge) Status() (api.ChargeStatus, error) {
	s, err := wb.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	switch s {
	case "disconnected":
		return api.StatusA, nil
	case "connected":
		return api.StatusB, nil
	case "charging":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *HeidelBridge) Enabled() (bool, error) {
	// HeidelBridge does not publish the enable/charging-current state
	// (0A is never reported), so the last set state is tracked locally
	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *HeidelBridge) Enable(enable bool) error {
	payload := "OFF"
	if enable {
		payload = "ON"
	}

	err := wb.enableS(payload)
	if err == nil {
		wb.enabled = enable
	}
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *HeidelBridge) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*HeidelBridge)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *HeidelBridge) MaxCurrentMillis(current float64) error {
	return wb.currentS(current)
}
