package charger

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// OpenWB configures generic charger and charge meter for an openWB loadpoint
type OpenWB struct {
	api.Charger
	currentPowerG func() (float64, error)
	totalEnergyG  func() (float64, error)
	currentsG     []func() (float64, error)
	authS         func(string) error
}

// go:generate go run ../cmd/tools/decorate.go -f decorateOpenWB -b *OpenWB -r api.Charger -t "api.ChargePhases,Phases1p3p,func(int) (error)" -t "api.Battery,SoC,func() (float64, error)"

// NewOpenWBFromConfig creates a new configurable charger
func NewOpenWBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
		ID          int
		Phases, DC  bool
	}{
		Topic:   openwb.RootTopic,
		Timeout: openwb.Timeout,
		ID:      1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb")

	return NewOpenWB(log, cc.Config, cc.ID, cc.Topic, cc.Phases, cc.DC, cc.Timeout)
}

// NewOpenWB creates a new configurable charger
func NewOpenWB(log *util.Logger, mqttconf mqtt.Config, id int, topic string, p1p3, dc bool, timeout time.Duration) (api.Charger, error) {
	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	// timeout handler
	timer := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/system/%s", topic, openwb.TimestampTopic), 1, timeout,
	).IntGetter()

	// getters
	boolG := func(topic string) func() (bool, error) {
		g := provider.NewMqtt(log, client, topic, 1, 0).BoolGetter()
		return func() (val bool, err error) {
			if val, err = g(); err == nil {
				_, err = timer()
			}
			return val, err
		}
	}

	floatG := func(topic string) func() (float64, error) {
		g := provider.NewMqtt(log, client, topic, 1, 0).FloatGetter()
		return func() (val float64, err error) {
			if val, err = g(); err == nil {
				_, err = timer()
			}
			return val, err
		}
	}

	// check if loadpoint configured
	configured := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ConfiguredTopic))
	if isConfigured, err := configured(); err != nil || !isConfigured {
		return nil, fmt.Errorf("openWB loadpoint %d is not configured", id)
	}

	// adapt plugged/charging to status
	pluggedG := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.PluggedTopic))
	chargingG := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargingTopic))
	statusG := provider.NewOpenWBStatusProvider(pluggedG, chargingG).StringGetter

	// remaining getters
	enabledG := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.EnabledTopic))

	// setters
	enableS := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/set/lp%d/%s", topic, id, openwb.EnabledTopic),
		1, timeout).WithPayload("${enable:%d}").BoolSetter("enable")
	maxcurrentS := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/set/lp%d/%s", topic, id, openwb.MaxCurrentTopic),
		1, timeout).IntSetter("maxcurrent")
	authS := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/set/chargepoint/%d/get/%s", topic, id, openwb.RfidTopic),
		1, timeout).StringSetter("rfid")

	// meter getters
	currentPowerG := floatG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargePowerTopic))
	totalEnergyG := floatG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargeTotalEnergyTopic))

	var currentsG []func() (float64, error)
	for i := 1; i <= 3; i++ {
		current := floatG(fmt.Sprintf("%s/lp/%d/%s%d", topic, id, openwb.CurrentTopic, i))
		currentsG = append(currentsG, current)
	}

	charger, err := NewConfigurable(statusG, enabledG, enableS, maxcurrentS)
	if err != nil {
		return nil, err
	}

	c := &OpenWB{
		Charger:       charger,
		currentPowerG: currentPowerG,
		totalEnergyG:  totalEnergyG,
		currentsG:     currentsG,
		authS:         authS,
	}

	// optional capabilities

	var phases func(int) error
	if p1p3 {
		phasesS := provider.NewMqtt(log, client,
			fmt.Sprintf("%s/set/isss/%s", topic, openwb.PhasesTopic),
			1, timeout).IntSetter("phases")

		phases = func(phases int) error {
			return phasesS(int64(phases))
		}
	}

	var soc func() (float64, error)
	if dc {
		soc = floatG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.VehicleSoCTopic))
	}

	return decorateOpenWB(c, phases, soc), nil
}

var _ api.Meter = (*OpenWB)(nil)

// CurrentPower implements the api.Meter interface
func (m *OpenWB) CurrentPower() (float64, error) {
	return m.currentPowerG()
}

var _ api.MeterEnergy = (*OpenWB)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *OpenWB) TotalEnergy() (float64, error) {
	return m.totalEnergyG()
}

var _ api.MeterCurrent = (*OpenWB)(nil)

// Currents implements the api.MeterCurrent interface
func (m *OpenWB) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, currentG := range m.currentsG {
		c, err := currentG()
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, c)
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Authorizer = (*OpenWB)(nil)

// Authorize implements the api.Authorizer interface
func (m *OpenWB) Authorize(key string) error {
	return m.authS(key)
}
