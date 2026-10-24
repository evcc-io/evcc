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
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Ambibox is the Ambibox ambiCHARGE Home charger implementation.
//
// Controlled over the box's MQTT (IPC) interface (vendor spec "Ipc 0.4.0",
// device type EV Charger). Topics below device/evCharger/<id>/:
//
//	limitChargePower  write  int W
//	stopCharge        write  bool
//	wakeUp            write  bool
//	sessionState      read   SESSION_SETUP ... CHARGE_LOOP, POST_CHARGE, PAUSED, STOPPED, ERROR
//	connected         read   bool
//	evConnected       read   bool
//	replugRequired    read   bool
//	controlMode       read   DISABLED, CONTROLLABLE, LIMITABLE
//	frequency         read   Hz, used as liveness signal
//	powerAc           read   int W, negative while charging
//	energyAcImport    read   Wh
//	energyAcExport    read   Wh
//	energyAcImportSession read Wh
//	soc               read   %
//	capacity          read   Wh
//	chargePowerMin    read   uint W
//	chargePowerMax    read   uint W
//	currentAc1..3     read   A
//	voltageAc1..3     read   V
//
// wakeUpBlocked is not part of the spec but published by firmware 0.4.8.
type Ambibox struct {
	log      *util.Logger
	client   ambiboxPublisher
	base     string
	current  float64
	minPower float64

	mu      sync.Mutex
	enabled *bool // nil until adopted from sessionState

	liveG          func() (float64, error)
	connectedG     func() (bool, error)
	evConnectedG   func() (bool, error)
	replugG        func() (bool, error)
	wakeBlockedG   func() (bool, error)
	sessionStateG  func() (string, error)
	controlModeG   func() (string, error)
	powerG         func() (float64, error)
	energyImpG     func() (float64, error)
	energyExpG     func() (float64, error)
	energySessImpG func() (float64, error)
	socG           func() (float64, error)
	capacityG      func() (float64, error)
	chargeMinG     func() (float64, error)
	chargeMaxG     func() (float64, error)
	currG          [3]func() (float64, error)
	voltG          [3]func() (float64, error)
}

// ambiboxPublisher is the one *mqtt.Client method Ambibox actually calls.
// Narrowed to an interface so tests can substitute a fake broker.
type ambiboxPublisher interface {
	Publish(topic string, retained bool, payload string)
}

// session states, as published on sessionState
const (
	ambiStateAuthorization = "AUTHORIZATION"
	ambiStateChargeLoop    = "CHARGE_LOOP"
	ambiStatePostCharge    = "POST_CHARGE"
	ambiStateStopped       = "STOPPED"
	ambiStateError         = "ERROR"
)

func init() {
	registry.Add("ambibox", NewAmbiboxFromConfig)
}

// NewAmbiboxFromConfig creates an Ambibox charger from configuration
func NewAmbiboxFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		ID          string
		MinPower    float64
		Timeout     time.Duration
	}{
		Topic:    "device/evCharger",
		MinPower: 600,
		Timeout:  30 * time.Second,
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

	return NewAmbibox(cc.Config, cc.Topic, cc.ID, cc.MinPower, cc.Timeout)
}

// NewAmbibox creates an Ambibox charger
func NewAmbibox(mqttconf mqtt.Config, topic, id string, minPower float64, timeout time.Duration) (*Ambibox, error) {
	log := util.NewLogger("ambibox")

	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	wb := &Ambibox{
		log:      log,
		client:   client,
		base:     fmt.Sprintf("%s/%s", topic, id),
		minPower: minPower,
	}

	boolG := func(sub string) (func() (bool, error), error) {
		return plugin.NewMqtt(log, client, wb.topic(sub), 0).BoolGetter()
	}
	floatG := func(sub string) (func() (float64, error), error) {
		return plugin.NewMqtt(log, client, wb.topic(sub), 0).FloatGetter()
	}
	stringG := func(sub string) (func() (string, error), error) {
		return plugin.NewMqtt(log, client, wb.topic(sub), 0).StringGetter()
	}

	// frequency is the only topic that streams on an interval rather than
	// strictly on change, so it is what liveness is judged on - a per-topic
	// timeout on any of the others would mark an idle but healthy box as
	// outdated.
	if wb.liveG, err = plugin.NewMqtt(log, client, wb.topic("frequency"), timeout).FloatGetter(); err != nil {
		return nil, err
	}
	if wb.connectedG, err = boolG("connected"); err != nil {
		return nil, err
	}
	if wb.evConnectedG, err = boolG("evConnected"); err != nil {
		return nil, err
	}
	if wb.replugG, err = boolG("replugRequired"); err != nil {
		return nil, err
	}
	if wb.wakeBlockedG, err = boolG("wakeUpBlocked"); err != nil {
		return nil, err
	}
	if wb.sessionStateG, err = stringG("sessionState"); err != nil {
		return nil, err
	}
	if wb.controlModeG, err = stringG("controlMode"); err != nil {
		return nil, err
	}
	if wb.powerG, err = floatG("powerAc"); err != nil {
		return nil, err
	}
	if wb.energyImpG, err = floatG("energyAcImport"); err != nil {
		return nil, err
	}
	if wb.energyExpG, err = floatG("energyAcExport"); err != nil {
		return nil, err
	}
	if wb.energySessImpG, err = floatG("energyAcImportSession"); err != nil {
		return nil, err
	}
	if wb.socG, err = floatG("soc"); err != nil {
		return nil, err
	}
	// optional: not published on firmware 0.4.8
	if wb.capacityG, err = floatG("capacity"); err != nil {
		return nil, err
	}
	if wb.chargeMinG, err = floatG("chargePowerMin"); err != nil {
		return nil, err
	}
	if wb.chargeMaxG, err = floatG("chargePowerMax"); err != nil {
		return nil, err
	}
	for i := range 3 {
		if wb.currG[i], err = floatG(fmt.Sprintf("currentAc%d", i+1)); err != nil {
			return nil, err
		}
		if wb.voltG[i], err = floatG(fmt.Sprintf("voltageAc%d", i+1)); err != nil {
			return nil, err
		}
	}

	// DISABLED means the box accepts neither limits nor setpoints, so evcc
	// would silently regulate against nothing. Worth saying out loud once.
	if mode, err := wb.controlModeG(); err == nil && mode == "DISABLED" {
		wb.log.WARN.Println("controlMode is DISABLED - the box will ignore all commands")
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

// chargeWatts converts a charging current into the charge power limit, based
// on measured voltage of the three-phase AC connection, held between the
// minimum and maximum charge power.
func (wb *Ambibox) chargeWatts(current float64) float64 {
	if current <= 0 {
		return 0
	}

	var sum float64
	for i := range wb.voltG {
		v := 230.0
		if m, err := wb.voltG[i](); err == nil && m > 0 {
			v = m
		}
		sum += v
	}

	power := current * sum

	minPower := wb.minPower
	if m, err := wb.chargeMinG(); err == nil {
		minPower = math.Max(minPower, m)
	}
	power = math.Max(power, minPower)

	if max, err := wb.chargeMaxG(); err == nil && max > 0 {
		power = math.Min(power, max)
	}

	return math.Max(0, math.Round(power))
}

func (wb *Ambibox) publishLimit(power float64) {
	wb.publish("limitChargePower", true, strconv.FormatInt(int64(power), 10))
}

// tryWake sends wakeUp unless the box itself says not to.
func (wb *Ambibox) tryWake() {
	if replug, err := wb.replugG(); err == nil && replug {
		return
	}
	if blocked, err := wb.wakeBlockedG(); err == nil && blocked {
		return
	}
	wb.publish("wakeUp", false, strconv.FormatBool(true))
}

// sessionActive reports whether the box has a session running or starting
// up. Without a vehicle it publishes no sessionState at all - that is a
// steady state, not a race, so it reads as no session rather than an error.
// While a vehicle is connected, a sessionState read error is reported rather
// than folded into "no session", so a caller adopting this at startup does
// not lock in a false guess while the retained message is still in flight.
func (wb *Ambibox) sessionActive() (bool, error) {
	s, err := wb.sessionStateG()
	if err != nil {
		if ev, evErr := wb.evConnectedG(); evErr == nil && ev {
			return false, err
		}
		return false, nil
	}
	if s == "" {
		return false, nil
	}
	return s != ambiStateStopped && s != ambiStateError && s != ambiStatePostCharge, nil
}

func (wb *Ambibox) assertLive() error {
	if _, err := wb.liveG(); err != nil {
		return api.ErrTimeout
	}
	return nil
}

// Status implements the api.Charger interface
func (wb *Ambibox) Status() (api.ChargeStatus, error) {
	if err := wb.assertLive(); err != nil {
		return api.StatusNone, err
	}

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

	s, err := wb.sessionStateG()
	if err != nil {
		return api.StatusNone, err
	}
	if s == ambiStateChargeLoop {
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
	if s, err := wb.sessionStateG(); err == nil && s == ambiStateAuthorization {
		return api.ReasonWaitingForAuthorization, nil
	}
	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (wb *Ambibox) Enabled() (bool, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if err := wb.assertLive(); err != nil {
		return false, err
	}

	if wb.enabled == nil {
		// Nothing was commanded yet - adopt whatever the box is doing, so a
		// restart mid-session does not report a charging box as disabled. A
		// getter error here (sessionState not delivered yet) is returned
		// without caching, so the next call retries adoption instead of
		// locking in a false "disabled".
		active, err := wb.sessionActive()
		if err != nil {
			return false, err
		}
		wb.enabled = &active
	}

	return *wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *Ambibox) Enable(enable bool) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.enabled = &enable

	if enable {
		wb.publishLimit(wb.chargeWatts(wb.current))
		wb.publish("stopCharge", false, strconv.FormatBool(false))
		wb.tryWake()
	} else {
		wb.publish("stopCharge", false, strconv.FormatBool(true))
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Ambibox) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Ambibox)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Ambibox) MaxCurrentMillis(current float64) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.current = current

	if wb.enabled != nil && *wb.enabled {
		wb.publishLimit(wb.chargeWatts(current))
	}

	return nil
}

var _ api.Meter = (*Ambibox)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Ambibox) CurrentPower() (float64, error) {
	if err := wb.assertLive(); err != nil {
		return 0, err
	}

	// the box reports charging as negative power, evcc expects it positive
	f, err := wb.powerG()
	if f == 0 {
		return 0, err
	}
	return -f, err
}

var _ api.MeterEnergy = (*Ambibox)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Ambibox) TotalEnergy() (float64, error) {
	if err := wb.assertLive(); err != nil {
		return 0, err
	}
	f, err := wb.energyImpG()
	return f / 1e3, err
}

var _ api.MeterReturnEnergy = (*Ambibox)(nil)

// ReturnEnergy implements the api.MeterReturnEnergy interface
func (wb *Ambibox) ReturnEnergy() (float64, error) {
	if err := wb.assertLive(); err != nil {
		return 0, err
	}
	f, err := wb.energyExpG()
	return f / 1e3, err
}

var _ api.ChargeRater = (*Ambibox)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Ambibox) ChargedEnergy() (float64, error) {
	if err := wb.assertLive(); err != nil {
		return 0, err
	}

	// Between sessions the box publishes no session energy at all, which
	// reads back as the same getter error as a genuine delivery problem.
	// sessionActive disambiguates: silently report zero only when there is
	// really no session to have a reading for, propagate anything else.
	f, err := wb.energySessImpG()
	if err != nil {
		if active, activeErr := wb.sessionActive(); activeErr != nil || active {
			return 0, err
		}
		return 0, nil
	}
	return f / 1e3, nil
}

var _ api.PhaseCurrents = (*Ambibox)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Ambibox) Currents() (float64, float64, float64, error) {
	if err := wb.assertLive(); err != nil {
		return 0, 0, 0, err
	}
	return wb.phaseValues(wb.currG)
}

var _ api.PhaseVoltages = (*Ambibox)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Ambibox) Voltages() (float64, float64, float64, error) {
	if err := wb.assertLive(); err != nil {
		return 0, 0, 0, err
	}
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
	if err := wb.assertLive(); err != nil {
		return 0, err
	}

	f, err := wb.socG()
	if err != nil {
		// only published while a vehicle is present
		return 0, api.ErrNotAvailable
	}
	return f, nil
}

var _ api.BatteryCapacity = (*Ambibox)(nil)

// Capacity implements the api.BatteryCapacity interface
func (wb *Ambibox) Capacity() (float64, error) {
	if err := wb.assertLive(); err != nil {
		return 0, err
	}

	f, err := wb.capacityG()
	if err != nil {
		return 0, err
	}
	return f / 1e3, nil
}

var _ api.Resurrector = (*Ambibox)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *Ambibox) WakeUp() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.tryWake()

	return nil
}
