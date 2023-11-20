package charger

import (
	"fmt"
	"strconv"
	"strings"
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
	current       int64
	enabled       bool
	statusG       func() (string, error)
	currentS      func(int64) error
	currentPowerG func() (float64, error)
	totalEnergyG  func() (float64, error)
	currentsG     []func() (float64, error)
	wakeupS       func(int64) error
	authS         func(string) error
}

//go:generate go run ../cmd/tools/decorate.go -f decorateOpenWB -b *OpenWB -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Battery,Soc,func() (float64, error)"

// NewOpenWBFromConfig creates a new configurable charger
func NewOpenWBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		mqtt.Config    `mapstructure:",squash"`
		Topic          string
		Timeout        time.Duration
		ID             int
		Phases1p3p, DC bool
	}{
		Topic:   openwb.RootTopic,
		Timeout: openwb.Timeout,
		ID:      1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb")

	return NewOpenWB(log, cc.Config, cc.ID, cc.Topic, cc.Phases1p3p, cc.DC, cc.Timeout)
}

// NewOpenWB creates a new configurable charger
func NewOpenWB(log *util.Logger, mqttconf mqtt.Config, id int, topic string, p1p3, dc bool, timeout time.Duration) (api.Charger, error) {
	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	// timeout handler
	h, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/system/%s", topic, openwb.TimestampTopic), timeout).StringGetter()
	if err != nil {
		return nil, err
	}
	to := provider.NewTimeoutHandler(h)

	mq := func(subtopic string) *provider.Mqtt {
		return provider.NewMqtt(log, client, fmt.Sprintf("%s/lp/%d/%s", topic, id, subtopic), 0)
	}

	// check if loadpoint configured
	configured, err := to.BoolGetter(mq(openwb.ConfiguredTopic))
	if err != nil {
		return nil, err
	}
	if isConfigured, err := configured(); err != nil || !isConfigured {
		return nil, fmt.Errorf("loadpoint %d is not configured", id)
	}

	// adapt plugged/charging to status
	pluggedG, err := to.BoolGetter(mq(openwb.PluggedTopic))
	if err != nil {
		return nil, err
	}
	chargingG, err := to.BoolGetter(mq(openwb.ChargingTopic))
	if err != nil {
		return nil, err
	}
	statusG := provider.NewOpenWBStatusProvider(pluggedG, chargingG).StringGetter

	// setters
	currentTopic := openwb.SlaveChargeCurrentTopic
	if id == 2 {
		// TODO remove after https://github.com/snaptec/openWB/issues/1757
		currentTopic = "Lp2" + openwb.SlaveChargeCurrentTopic
	}
	currentS, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/set/isss/%s", topic, currentTopic),
		timeout).WithRetained().IntSetter("current")
	if err != nil {
		return nil, err
	}

	cpTopic := openwb.SlaveCPInterruptTopic
	if id == 2 {
		// TODO remove after https://github.com/snaptec/openWB/issues/1757
		cpTopic = strings.TrimSuffix(cpTopic, "1") + "2"
	}
	wakeupS, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/set/isss/%s", topic, cpTopic),
		timeout).WithRetained().IntSetter("cp")
	if err != nil {
		return nil, err
	}

	authS, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/set/chargepoint/%d/set/%s", topic, id, openwb.RfidTopic),
		timeout).WithRetained().StringSetter("rfid")
	if err != nil {
		return nil, err
	}

	// meter getters
	currentPowerG, err := to.FloatGetter(mq(openwb.ChargePowerTopic))
	if err != nil {
		return nil, err
	}
	totalEnergyG, err := to.FloatGetter(mq(openwb.ChargeTotalEnergyTopic))
	if err != nil {
		return nil, err
	}

	var currentsG []func() (float64, error)
	for i := 1; i <= 3; i++ {
		current, err := to.FloatGetter(mq(openwb.CurrentTopic + strconv.Itoa(i)))
		if err != nil {
			return nil, err
		}
		currentsG = append(currentsG, current)
	}

	c := &OpenWB{
		currentS:      currentS,
		statusG:       statusG,
		currentPowerG: currentPowerG,
		totalEnergyG:  totalEnergyG,
		currentsG:     currentsG,
		wakeupS:       wakeupS,
		authS:         authS,
	}

	// heartbeat
	heartbeatS, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/set/isss/%s", topic, openwb.SlaveHeartbeatTopic),
		timeout).WithRetained().IntSetter("heartbeat")
	if err != nil {
		return nil, err
	}

	go func() {
		for range time.Tick(openwb.HeartbeatInterval) {
			if err := heartbeatS(1); err != nil {
				log.ERROR.Printf("heartbeat: %v", err)
			}
		}
	}()

	// optional capabilities

	var phases func(int) error
	if p1p3 {
		phasesTopic := openwb.SlavePhasesTopic
		if id == 2 {
			// TODO remove after https://github.com/snaptec/openWB/issues/1757
			phasesTopic += "Lp2"
		}
		phasesS, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/set/isss/%s", topic, phasesTopic),
			timeout).WithRetained().IntSetter("phases")
		if err != nil {
			return nil, err
		}

		phases = func(phases int) error {
			return phasesS(int64(phases))
		}
	}

	var soc func() (float64, error)
	if dc {
		soc, err = to.FloatGetter(mq(openwb.VehicleSocTopic))
		if err != nil {
			return nil, err
		}
	}

	return decorateOpenWB(c, phases, soc), nil
}

func (m *OpenWB) Enable(enable bool) error {
	var current int64
	if enable {
		current = m.current
	}

	err := m.currentS(current)
	if err == nil {
		m.enabled = enable
	}

	return err
}

func (m *OpenWB) Enabled() (bool, error) {
	return verifyEnabled(m, m.enabled)
}

func (m *OpenWB) Status() (api.ChargeStatus, error) {
	status, err := m.statusG()
	if err != nil {
		return api.StatusNone, err
	}
	return api.ChargeStatusString(status)
}

func (m *OpenWB) MaxCurrent(current int64) error {
	err := m.currentS(current)
	if err == nil {
		m.current = current
	}
	return err
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

var _ api.PhaseCurrents = (*OpenWB)(nil)

// Currents implements the api.PhaseCurrents interface
func (m *OpenWB) Currents() (float64, float64, float64, error) {
	var res []float64
	for _, currentG := range m.currentsG {
		c, err := currentG()
		if err != nil {
			return 0, 0, 0, err
		}

		res = append(res, c)
	}

	return res[0], res[1], res[2], nil
}

var _ api.Authorizer = (*OpenWB)(nil)

// Authorize implements the api.Authorizer interface
func (m *OpenWB) Authorize(key string) error {
	return m.authS(key)
}

var _ api.Resurrector = (*OpenWB)(nil)

// WakeUp implements the api.Resurrector interface
func (m *OpenWB) WakeUp() error {
	return m.wakeupS(1)
}
