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
		ID          int
	}{
		Topic:   warpRootTopic,
		Timeout: warpTimeout,
		ID:      1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewWarp(cc.Config, cc.ID, cc.Topic, cc.Timeout)
}

// Warp configures generic charger and charge meter for an Warp loadpoint
type Warp struct {
	evse        string
	client      *mqtt.Client
	enableS     func(bool) error
	enabledG    func() (string, error)
	statusG     func() (string, error)
	maxcurrentS func(int64) error
}

// NewWarp creates a new configurable charger
func NewWarp(mqttconf mqtt.Config, id int, topic string, timeout time.Duration) (*Warp, error) {
	log := util.NewLogger("warp")

	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	evse := fmt.Sprintf("%s/evse", topic)

	enableS := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/auto_start_charging", evse),
		`{ "auto_start_charging": ${enable} }`, 1, 0,
	).BoolSetter("enabled")

	enabledG := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/auto_start_charging", evse), "", 1, 0).StringGetter()

	statusG := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/state", evse), "", 1, 0).StringGetter()

	maxcurrentS := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/max_charging_current", evse),
		`{ "max_current_configured": ${maxcurrent} }`, 1, 0,
	).IntSetter("maxcurrent")

	m := &Warp{
		evse:        evse,
		client:      client,
		enableS:     enableS,
		enabledG:    enabledG,
		statusG:     statusG,
		maxcurrentS: maxcurrentS,
	}

	return m, nil

}

type warpAutoCharging struct {
	AutoStartCharging bool `json:"auto_start_charging"`
}

// Enable implements the api.Charger interface
func (m *Warp) Enable(enable bool) error {
	return m.enableS(enable)
}

// Enabled implements the api.Charger interface
func (m *Warp) Enabled() (bool, error) {
	var res warpAutoCharging

	s, err := m.enabledG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.AutoStartCharging, err
}

type warpStatus struct {
	Iec61851State          int64 `json:"iec61851_state"`
	VehicleState           int64 `json:"vehicle_state"`
	ContactorState         int64 `json:"contactor_state"`
	ContactorError         int64 `json:"contactor_error"`
	AllowedChargingCurrent int64 `json:"allowed_charging_current"`
	ErrorState             int64 `json:"error_state"`
	LockState              int64 `json:"lock_state"`
	TimeSinceStateChange   int64 `json:"time_since_state_change"`
	Uptime                 int64 `json:"uptime"`
}

// Status implements the api.Charger interface
func (m *Warp) Status() (api.ChargeStatus, error) {
	var status warpStatus

	s, err := m.statusG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &status)
	}

	if err != nil {
		return api.StatusNone, err
	}

	res := api.StatusA
	if status.VehicleState > 0 {

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

// // CurrentPower implements the Meter.CurrentPower interface
// func (m *Warp) CurrentPower() (float64, error) {
// 	return m.currentPowerG()
// }

// // TotalEnergy implements the Meter.TotalEnergy interface
// func (m *Warp) TotalEnergy() (float64, error) {
// 	return m.totalEnergyG()
// }

// // Currents implements the Meter.Currents interface
// func (m *Warp) Currents() (float64, float64, float64, error) {
// 	var currents []float64
// 	for _, currentG := range m.currentsG {
// 		c, err := currentG()
// 		if err != nil {
// 			return 0, 0, 0, err
// 		}

// 		currents = append(currents, c)
// 	}

// 	return currents[0], currents[1], currents[2], nil
// }
