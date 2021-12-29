package charger

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

// Warp configures generic charger and charge meter for an Warp loadpoint
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

//go:generate go run ../cmd/tools/decorate.go -f decorateWarp -b *Warp -r api.Charger -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewWarpFromConfig creates a new configurable charger
func NewWarpFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
		UseMeter    interface{}
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

	if cc.UseMeter != nil {
		// TODO remove
		util.NewLogger("warp").WARN.Println("usemeter is deprecated and will be removed in a future release")
	}

	detectPro := provider.NewMqtt(wb.log, wb.client,
		fmt.Sprintf("%s/evse/low_level_state", wb.root), 1, cc.Timeout,
	).StringGetter()

	var isPro bool
	if state, err := detectPro(); err == nil {
		var res warp.LowLevelState
		if err := json.Unmarshal([]byte(state), &res); err != nil {
			return nil, err
		}

		isPro = len(res.AdcValues) > 2
	}

	var currents func() (float64, float64, float64, error)
	if isPro {
		currents = wb.currents
	}

	return decorateWarp(wb, currents), err
}

// NewWarp creates a new configurable charger
func NewWarp(mqttconf mqtt.Config, topic string, timeout time.Duration) (*Warp, error) {
	log := util.NewLogger("warp")

	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	m := &Warp{
		log:    log,
		root:   topic,
		client: client,
	}

	// timeout handler
	timer := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/state", topic), 1, timeout,
	).StringGetter()

	stringG := func(topic string) func() (string, error) {
		g := provider.NewMqtt(log, client, topic, 1, 0).StringGetter()
		return func() (val string, err error) {
			if val, err = g(); err == nil {
				_, err = timer()
			}
			return val, err
		}
	}

	m.enabledG = stringG(fmt.Sprintf("%s/evse/auto_start_charging", topic))
	m.statusG = stringG(fmt.Sprintf("%s/evse/state", topic))
	m.meterG = stringG(fmt.Sprintf("%s/meter/state", topic))
	m.meterDetailsG = stringG(fmt.Sprintf("%s/meter/detailed_values", topic))

	m.enableS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/auto_start_charging_update", topic), 1, 0).
		WithPayload(`{ "auto_start_charging": ${enable} }`).
		BoolSetter("enable")

	m.maxcurrentS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/current_limit", topic), 1, 0).
		WithPayload(`{ "current": ${maxcurrent} }`).
		IntSetter("maxcurrent")

	return m, nil
}

// Enable implements the api.Charger interface
func (m *Warp) Enable(enable bool) error {
	// set auto_start_charging
	if err := m.enableS(enable); err != nil {
		return err
	}

	// trigger start/stop
	action := "stop_charging"
	if enable {
		action = "start_charging"
	}

	topic := fmt.Sprintf("%s/%s/%s", m.root, "evse", action)

	err := m.client.Publish(topic, false, "null")
	if err == nil {
		m.enabled = enable
	}

	return err
}

func (m *Warp) status() (warp.Status, error) {
	var res warp.Status

	s, err := m.statusG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res, err
}

// autostart reads the enabled state from charger
// use function instead of jq to honor evse/state updates
func (m *Warp) autostart() (bool, error) {
	var res struct {
		AutoStartCharging bool `json:"auto_start_charging"`
	}

	s, err := m.enabledG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.AutoStartCharging, err
}

// isEnabled reads enabled status from mqtt
func (m *Warp) isEnabled() (bool, error) {
	enabled, err := m.autostart()

	var status warp.Status
	if err == nil {
		status, err = m.status()
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
func (m *Warp) Enabled() (bool, error) {
	enabled, err := m.isEnabled()

	if err == nil && enabled != m.enabled {
		start := time.Now()

		// retry to avoid out of sync errors in case of slow warp updates
		for time.Since(start) <= 2*time.Second {
			if enabled, err = m.isEnabled(); err != nil {
				break
			}

			if enabled == m.enabled {
				break
			}

			time.Sleep(50 * time.Millisecond)
		}
	}

	return enabled, err
}

// Status implements the api.Charger interface
func (m *Warp) Status() (api.ChargeStatus, error) {
	var status warp.Status

	s, err := m.statusG()
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
func (m *Warp) MaxCurrent(current int64) error {
	return m.maxcurrentS(1000 * current)
}

var _ api.ChargerEx = (*Warp)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (m *Warp) MaxCurrentMillis(current float64) error {
	return m.maxcurrentS(int64(1000 * current))
}

var _ api.Meter = (*Warp)(nil)

// CurrentPower implements the api.Meter interface
func (m *Warp) CurrentPower() (float64, error) {
	var res warp.PowerStatus

	s, err := m.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.Power, err
}

var _ api.MeterEnergy = (*Warp)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *Warp) TotalEnergy() (float64, error) {
	var res warp.PowerStatus

	s, err := m.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.EnergyAbs, err
}

// currents implements the api.MeterCurrrents interface
func (m *Warp) currents() (float64, float64, float64, error) {
	var res []float64

	s, err := m.meterDetailsG()
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
