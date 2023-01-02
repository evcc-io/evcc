package charger

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	v1 "github.com/evcc-io/evcc/charger/warp/v1"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

// Warp is the Warp charger implementation
type Warp struct {
	log           *util.Logger
	root          string
	client        *mqtt.Client
	enabledG      func() (string, error)
	statusG       func() (string, error)
	meterG        func() (string, error)
	meterDetailsG func() (string, error)
	enableS       func(bool) error
	maxcurrentS   func(int64) error
	enabled       bool // cache
}

func init() {
	registry.Add("warp", NewWarpFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateWarp -b *Warp -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewWarpFromConfig creates a new configurable charger
func NewWarpFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
	}{
		Topic:   warp.RootTopic,
		Timeout: warp.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarp(cc.Config, cc.Topic, cc.Timeout)
	if err != nil {
		return nil, err
	}

	hasMeter := wb.hasMeter()

	var currentPower, totalEnergy func() (float64, error)
	if hasMeter {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
	}

	var currents func() (float64, float64, float64, error)
	if hasMeter && wb.hasCurrents() {
		currents = wb.currents
	}

	return decorateWarp(wb, currentPower, totalEnergy, currents), err
}

// NewWarp creates a new configurable charger
func NewWarp(mqttconf mqtt.Config, topic string, timeout time.Duration) (*Warp, error) {
	log := util.NewLogger("warp")

	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	wb := &Warp{
		log:    log,
		root:   topic,
		client: client,
	}

	// timeout handler
	to := provider.NewTimeoutHandler(provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/state", topic), timeout,
	).StringGetter())

	stringG := func(topic string) func() (string, error) {
		g := provider.NewMqtt(log, client, topic, 0).StringGetter()
		return to.StringGetter(g)
	}

	wb.enabledG = stringG(fmt.Sprintf("%s/evse/auto_start_charging", topic))
	wb.statusG = stringG(fmt.Sprintf("%s/evse/state", topic))
	wb.meterG = stringG(fmt.Sprintf("%s/meter/state", topic))
	wb.meterDetailsG = stringG(fmt.Sprintf("%s/meter/detailed_values", topic))

	wb.enableS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/auto_start_charging_update", topic), 0).
		WithPayload(`{ "auto_start_charging": ${enable} }`).
		BoolSetter("enable")

	wb.maxcurrentS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/current_limit", topic), 0).
		WithPayload(`{ "current": ${maxcurrent} }`).
		IntSetter("maxcurrent")

	return wb, nil
}

func (wb *Warp) hasMeter() bool {
	topic := fmt.Sprintf("%s/meter/state", wb.root)

	if state, err := provider.NewMqtt(wb.log, wb.client, topic, 0).StringGetter()(); err == nil {
		var res v1.MeterState
		if err := json.Unmarshal([]byte(state), &res); err == nil {
			return res.State == 2 || len(res.PhasesConnected) > 0
		}
	}

	return false
}

func (wb *Warp) hasCurrents() bool {
	topic := fmt.Sprintf("%s/meter/detailed_values", wb.root)

	if state, err := provider.NewMqtt(wb.log, wb.client, topic, 0).StringGetter()(); err == nil {
		var res []float64
		if err := json.Unmarshal([]byte(state), &res); err == nil {
			return len(res) > 5
		}
	}

	return false
}

// Enable implements the api.Charger interface
func (wb *Warp) Enable(enable bool) error {
	// set auto_start_charging
	if err := wb.enableS(enable); err != nil {
		return err
	}

	// trigger start/stop
	action := "stop_charging"
	if enable {
		action = "start_charging"
	}

	topic := fmt.Sprintf("%s/%s/%s", wb.root, "evse", action)

	err := wb.client.Publish(topic, false, "null")
	if err == nil {
		wb.enabled = enable
	}

	return err
}

func (wb *Warp) status() (v1.EvseState, error) {
	var res v1.EvseState

	s, err := wb.statusG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res, err
}

// autostart reads the enabled state from charger
// use function instead of jq to honor evse/state updates
func (wb *Warp) autostart() (bool, error) {
	var res struct {
		AutoStartCharging bool `json:"auto_start_charging"`
	}

	s, err := wb.enabledG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.AutoStartCharging, err
}

// isEnabled reads enabled status from mqtt
func (wb *Warp) isEnabled() (bool, error) {
	enabled, err := wb.autostart()

	var status v1.EvseState
	if err == nil {
		status, err = wb.status()
	}

	if enabled {
		// check that charge_release is not blocked
		enabled = status.ChargeRelease != 2
	} else {
		// check that vehicle is really not charging
		enabled = status.VehicleState == 2
	}

	return enabled, err
}

// Enabled implements the api.Charger interface
func (wb *Warp) Enabled() (bool, error) {
	enabled, err := wb.isEnabled()

	if err == nil && enabled != wb.enabled {
		start := time.Now()

		// retry to avoid out of sync errors in case of slow warp updates
		for time.Since(start) <= 2*time.Second {
			if enabled, err = wb.isEnabled(); err != nil {
				break
			}

			if enabled == wb.enabled {
				break
			}

			time.Sleep(50 * time.Millisecond)
		}
	}

	return enabled, err
}

// Status implements the api.Charger interface
func (wb *Warp) Status() (api.ChargeStatus, error) {
	var status v1.EvseState

	s, err := wb.statusG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &status)
	}

	res := api.StatusNone
	switch status.VehicleState {
	case 0:
		res = api.StatusA
	case 1:
		res = api.StatusB
	case 2:
		res = api.StatusC
	default:
		if err == nil {
			err = fmt.Errorf("invalid status: %d", status.VehicleState)
		}
	}

	return res, err
}

// MaxCurrent implements the api.Charger interface
func (wb *Warp) MaxCurrent(current int64) error {
	return wb.maxcurrentS(1000 * current)
}

var _ api.ChargerEx = (*Warp)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Warp) MaxCurrentMillis(current float64) error {
	return wb.maxcurrentS(int64(1000 * current))
}

// CurrentPower implements the api.Meter interface
func (wb *Warp) currentPower() (float64, error) {
	var res v1.MeterState

	s, err := wb.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.Power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Warp) totalEnergy() (float64, error) {
	var res v1.MeterState

	s, err := wb.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.EnergyAbs, err
}

// currents implements the api.MeterCurrrents interface
func (wb *Warp) currents() (float64, float64, float64, error) {
	var res []float64

	s, err := wb.meterDetailsG()
	if err == nil {
		if err = json.Unmarshal([]byte(s), &res); err == nil {
			if len(res) > 5 {
				return res[3], res[4], res[5], nil
			}

			err = errors.New("invalid length")
		}
	}

	return 0, 0, 0, err
}
