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
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Ambibox is the Ambibox ambiCHARGE Home charger implementation.
// It talks to the wallbox via the standardized Ambibox MQTT device interface
// (Ipc 0.4.0-pre, section "2.4 EV Charger"). All topics live below the prefix
// device/evCharger/{id}/…: evcc subscribes to the published state and publishes
// to the control topics.
type Ambibox struct {
	log    *util.Logger
	client *mqtt.Client
	base   string

	// state getters (device publishes, evcc subscribes)
	connectedG    func() (bool, error)
	evConnectedG  func() (bool, error)
	replugG       func() (bool, error)
	sessionStateG func() (string, error)
	powerG        func() (float64, error)
	energyImpG    func() (float64, error)
	energyExpG    func() (float64, error)
	energySessG   func() (float64, error)
	socG          func() (float64, error)
	phasesG       func() (int64, error)
	currG         [3]func() (float64, error)
	voltG         [3]func() (float64, error)
	targetPowerG  func() (float64, error) // read-back of the published control setpoint

	// last requested current in A, restored when charging is (re-)enabled
	current float64
}

func init() {
	registry.Add("ambibox", NewAmbiboxFromConfig)
}

// NewAmbiboxFromConfig creates an Ambibox charger from configuration
func NewAmbiboxFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		ID          string
		Timeout     time.Duration
	}{
		Topic:   "device/evCharger",
		Timeout: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ID == "" {
		return nil, fmt.Errorf("missing id")
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	return NewAmbibox(cc.Config, cc.Topic, cc.ID, cc.Timeout)
}

// NewAmbibox creates an Ambibox charger
func NewAmbibox(mqttconf mqtt.Config, topic, id string, timeout time.Duration) (*Ambibox, error) {
	log := util.NewLogger("ambibox")

	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	wb := &Ambibox{
		log:    log,
		client: client,
		base:   fmt.Sprintf("%s/%s", topic, id),
	}

	// core status topics use the configured timeout, slow-changing topics use none
	boolG := func(sub string, to time.Duration) (func() (bool, error), error) {
		return plugin.NewMqtt(log, client, wb.topic(sub), to).BoolGetter()
	}
	floatG := func(sub string, to time.Duration) (func() (float64, error), error) {
		return plugin.NewMqtt(log, client, wb.topic(sub), to).FloatGetter()
	}

	if wb.connectedG, err = boolG("connected", timeout); err != nil {
		return nil, err
	}
	if wb.evConnectedG, err = boolG("evConnected", timeout); err != nil {
		return nil, err
	}
	if wb.replugG, err = boolG("replugRequired", 0); err != nil {
		return nil, err
	}
	if wb.sessionStateG, err = plugin.NewMqtt(log, client, wb.topic("sessionState"), timeout).StringGetter(); err != nil {
		return nil, err
	}
	if wb.powerG, err = floatG("powerAc", timeout); err != nil {
		return nil, err
	}
	if wb.energyImpG, err = floatG("energyAcImport", 0); err != nil {
		return nil, err
	}
	if wb.energyExpG, err = floatG("energyAcExport", 0); err != nil {
		return nil, err
	}
	if wb.energySessG, err = floatG("energyAcImportSession", 0); err != nil {
		return nil, err
	}
	if wb.socG, err = floatG("soc", 0); err != nil {
		return nil, err
	}
	if wb.targetPowerG, err = floatG("targetPower", timeout); err != nil {
		return nil, err
	}
	if wb.phasesG, err = plugin.NewMqtt(log, client, wb.topic("numberPhases"), 0).IntGetter(); err != nil {
		return nil, err
	}
	for i := range 3 {
		if wb.currG[i], err = floatG(fmt.Sprintf("currentAc%d", i+1), 0); err != nil {
			return nil, err
		}
		if wb.voltG[i], err = floatG(fmt.Sprintf("voltageAc%d", i+1), 0); err != nil {
			return nil, err
		}
	}

	return wb, nil
}

func (wb *Ambibox) topic(sub string) string {
	return wb.base + "/" + sub
}

// publish publishes a plain scalar payload
func (wb *Ambibox) publish(sub string, retained bool, payload string) {
	wb.client.Publish(wb.topic(sub), retained, payload)
}

// targetWatts converts a charging current (AC) into the Ambibox targetPower (DC)
// setpoint (negative = charge), based on measured voltage and active phases.
func (wb *Ambibox) targetWatts(current float64) float64 {
	phases := 3
	if p, err := wb.phasesG(); err == nil && p >= 1 && p <= 3 {
		phases = int(p)
	}

	var sum float64
	for i := range phases {
		v := 230.0
		if m, err := wb.voltG[i](); err == nil && m > 0 {
			v = m
		}
		sum += v
	}

	return -current * sum
}

// setTargetPhaseCurrent publishes the targetPower setpoint for the given current.
// Published retained so it survives reconnects and can be read back via Enabled.
func (wb *Ambibox) setTargetPhaseCurrent(current float64) {
	power := math.Round(wb.targetWatts(current))
	wb.publish("targetPower", true, strconv.FormatInt(int64(power), 10))
}

// Status implements the api.Charger interface
func (wb *Ambibox) Status() (api.ChargeStatus, error) {
	if connected, err := wb.connectedG(); err != nil {
		return api.StatusNone, err
	} else if !connected {
		return api.StatusNone, api.ErrTimeout
	}

	ev, err := wb.evConnectedG()
	if err != nil {
		return api.StatusNone, err
	}
	if !ev {
		return api.StatusA, nil
	}

	// vehicle connected: distinguish charging (C) from connected-only (B)
	if s, err := wb.sessionStateG(); err == nil && s == "CHARGE_LOOP" {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

var _ api.StatusReasoner = (*Ambibox)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *Ambibox) StatusReason() (api.Reason, error) {
	if replug, err := wb.replugG(); err == nil && replug {
		return api.ReasonDisconnectRequired, nil
	}
	if s, err := wb.sessionStateG(); err == nil && s == "AUTHORIZATION" {
		return api.ReasonWaitingForAuthorization, nil
	}
	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (wb *Ambibox) Enabled() (bool, error) {
	power, err := wb.targetPowerG()
	return power != 0, err
}

// Enable implements the api.Charger interface
func (wb *Ambibox) Enable(enable bool) error {
	var current float64
	if enable {
		current = wb.current
	}
	wb.setTargetPhaseCurrent(current)

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Ambibox) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Ambibox)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Ambibox) MaxCurrentMillis(current float64) error {
	wb.setTargetPhaseCurrent(current)
	wb.current = current

	return nil
}

var _ api.Meter = (*Ambibox)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Ambibox) CurrentPower() (float64, error) {
	return wb.powerG()
}

var _ api.MeterEnergy = (*Ambibox)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Ambibox) TotalEnergy() (float64, error) {
	f, err := wb.energyImpG()
	return f / 1e3, err
}

var _ api.MeterReturnEnergy = (*Ambibox)(nil)

// ReturnEnergy implements the api.MeterReturnEnergy interface
func (wb *Ambibox) ReturnEnergy() (float64, error) {
	f, err := wb.energyExpG()
	return f / 1e3, err
}

var _ api.ChargeRater = (*Ambibox)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Ambibox) ChargedEnergy() (float64, error) {
	f, err := wb.energySessG()
	return f / 1e3, err
}

var _ api.PhaseCurrents = (*Ambibox)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Ambibox) Currents() (float64, float64, float64, error) {
	return wb.phaseValues(wb.currG)
}

var _ api.PhaseVoltages = (*Ambibox)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Ambibox) Voltages() (float64, float64, float64, error) {
	return wb.phaseValues(wb.voltG)
}

func (wb *Ambibox) phaseValues(g [3]func() (float64, error)) (float64, float64, float64, error) {
	var res [3]float64
	for i, f := range g {
		v, err := f()
		if err != nil {
			return 0, 0, 0, err
		}
		res[i] = v
	}
	return res[0], res[1], res[2], nil
}

var _ api.Battery = (*Ambibox)(nil)

// Soc implements the api.Battery interface
func (wb *Ambibox) Soc() (float64, error) {
	return wb.socG()
}

var _ api.Resurrector = (*Ambibox)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *Ambibox) WakeUp() error {
	wb.publish("wakeUp", false, strconv.FormatBool(true))
	return nil
}
