package charge

import (
	"fmt"
	"time"

	"github.com/andig/evcc-config/registry"
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

	log := util.NewLogger("warp")

	return NewWarp(log, cc.Config, cc.ID, cc.Topic, cc.Timeout)
}

// Warp configures generic charger and charge meter for an Warp loadpoint
type Warp struct {
	evse          string
	client        *mqtt.Client
	statusG       func() (int64, error)
	maxcurrentS   func(int64) error
	currentPowerG func() (float64, error)
	totalEnergyG  func() (float64, error)
}

// NewWarp creates a new configurable charger
func NewWarp(log *util.Logger, mqttconf mqtt.Config, id int, topic string, timeout time.Duration) (*Warp, error) {
	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	evse := fmt.Sprintf("%s/evse", topic)

	statusG := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/iec61851_state", evse), "", 1, 0).IntGetter()

	maxcurrentS := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/allowed_charging_current", evse), "", 1, timeout).IntSetter("maxcurrent")

	// // adapt plugged/charging to status
	// plugged := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, warp.PluggedTopic))
	// charging := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, warp.ChargingTopic))
	// status := provider.NewWarpStatusProvider(plugged, charging).StringGetter

	// // remaining getters
	// enabled := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, warp.EnabledTopic))

	// // setters
	// enable := provider.NewMqtt(log, client,
	// 	fmt.Sprintf("%s/set/lp%d/%s", topic, id, warp.EnabledTopic),
	// 	"", 1, timeout).BoolSetter("enable")
	// maxcurrent := provider.NewMqtt(log, client,
	// 	fmt.Sprintf("%s/set/lp%d/%s", topic, id, warp.MaxCurrentTopic),
	// 	"", 1, timeout).IntSetter("maxcurrent")

	// // meter getters
	// currentPowerG := floatG(fmt.Sprintf("%s/lp/%d/%s", topic, id, warp.ChargePowerTopic))
	// totalEnergyG := floatG(fmt.Sprintf("%s/lp/%d/%s", topic, id, warp.ChargeTotalEnergyTopic))

	// var currentsG []func() (float64, error)
	// for i := 1; i <= 3; i++ {
	// 	current := floatG(fmt.Sprintf("%s/lp/%d/%s%d", topic, id, warp.CurrentTopic, i))
	// 	currentsG = append(currentsG, current)
	// }

	// charger, err := NewConfigurable(status, enabled, enable, maxcurrent)
	// if err != nil {
	// 	return nil, err
	// }

	// res := &Warp{
	// 	Charger:       charger,
	// }

	m := &Warp{
		evse:    evse,
		client:  client,
		statusG: statusG,
		// currentPowerG: currentPowerG,
		// totalEnergyG:  totalEnergyG,
		// currentsG:     currentsG,
	}

	return m, nil

}

func (m *Warp) Enable(enable bool) error {
	enableS := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/start_charging", evse), "", 1, timeout).BoolSetter("enable")
}

// Status implements the Meter.Status interface
func (m *Warp) Status() (api.ChargeStatus, error) {
	val, err := m.statusG()
	if err != nil {
		return api.StatusF, err
	}

	var res api.ChargeStatus
	switch val {
	case 0:
		res = api.StatusA
	case 1:
		res = api.StatusB
	case 2:
		res = api.StatusC
	case 3:
		res = api.StatusD
	case 4:
		res = api.StatusF
	default:
		err = fmt.Errorf("invalid status: %d", val)
	}

	return res, err
}

func (m *Warp) MaxCurrent(current int64) error {
	return m.maxcurrentS(1000 * current)
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Warp) CurrentPower() (float64, error) {
	return m.currentPowerG()
}

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
