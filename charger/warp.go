package charger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/provider/mqtt"
	"github.com/andig/evcc/util"
)

func init() {
	registry.Add("warp", NewWarpFromConfig)
}

const (
	warpRootTopic = "warp"
	warpTimeout   = 30 * time.Second
)

// NewWarpFromConfig creates a new configurable charger
func NewWarpFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
		UseMeter    bool
	}{
		Topic:   warpRootTopic,
		Timeout: warpTimeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	m, err := NewWarp(cc.Config, cc.Topic, cc.Timeout)
	if err != nil {
		return nil, err
	}

	var currentPower func() (float64, error)
	var totalEnergy func() (float64, error)
	if cc.UseMeter {
		currentPower = m.currentPower
		totalEnergy = m.totalEnergy
	}

	return decorateWarp(m, currentPower, totalEnergy), err
}

// Warp configures generic charger and charge meter for an Warp loadpoint
type Warp struct {
	log         *util.Logger
	root        string
	client      *mqtt.Client
	enabledG    func() (string, error)
	statusG     func() (string, error)
	meterG      func() (string, error)
	enableS     func(bool) error
	maxcurrentS func(int64) error
	enabled     bool // cache
}

//go:generate go run ../cmd/tools/decorate.go -p charger -f decorateWarp -o warp_decorators -b *Warp -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)"

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
		fmt.Sprintf("%s/evse/state", topic), "", 1, timeout,
	).StringGetter()

	stringG := func(topic string) func() (string, error) {
		g := provider.NewMqtt(log, client, topic, "", 1, 0).StringGetter()
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

	m.enableS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/auto_start_charging_update", topic),
		`{ "auto_start_charging": ${enable} }`, 1, 0,
	).BoolSetter("enable")

	m.maxcurrentS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/current_limit", topic),
		`{ "current": ${maxcurrent} }`, 1, 0,
	).IntSetter("maxcurrent")

	return m, nil
}

type warpStatus struct {
	Iec61851State          int   `json:"iec61851_state"`
	VehicleState           int   `json:"vehicle_state"`
	ChargeRelease          int   `json:"charge_release"`
	ContactorState         int   `json:"contactor_state"`
	ContactorError         int   `json:"contactor_error"`
	AllowedChargingCurrent int64 `json:"allowed_charging_current"`
	ErrorState             int   `json:"error_state"`
	LockState              int   `json:"lock_state"`
	TimeSinceStateChange   int64 `json:"time_since_state_change"`
	Uptime                 int64 `json:"uptime"`
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

	err := m.client.Publish(topic, true, "null")
	if err == nil {
		m.enabled = enable
	}

	return err
}

func (m *Warp) status() (warpStatus, error) {
	var res warpStatus

	s, err := m.statusG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res, err
}

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

	var status warpStatus
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
	var status warpStatus

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

// MaxCurrentMillis implements the api.ChargerEx interface
func (m *Warp) MaxCurrentMillis(current float64) error {
	return m.maxcurrentS(int64(1000 * current))
}

type powerStatus struct {
	Power     float64 `json:"power"`
	EnergyRel float64 `json:"energy_rel"`
	EnergyAbs float64 `json:"energy_abs"`
}

// currentPower implements the Meter.CurrentPower interface
func (m *Warp) currentPower() (float64, error) {
	var res powerStatus

	s, err := m.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.Power, err
}

// totalEnergy implements the Meter.TotalEnergy interface
func (m *Warp) totalEnergy() (float64, error) {
	var res powerStatus

	s, err := m.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.EnergyAbs, err
}
