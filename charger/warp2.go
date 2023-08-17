package charger

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

// Warp2 is the Warp charger v2 firmware implementation
type Warp2 struct {
	log           *util.Logger
	client        *mqtt.Client
	features      []string
	maxcurrentG   func() (string, error)
	statusG       func() (string, error)
	meterG        func() (string, error)
	meterDetailsG func() (string, error)
	chargeG       func() (string, error)
	userconfigG   func() (string, error)
	maxcurrentS   func(int64) error
	// emConfigG     func() (string, error)
	emStateG func() (string, error)
	phasesS  func(int64) error
	current  int64
}

func init() {
	registry.Add("warp2", NewWarp2FromConfig)
	registry.Add("warp-fw2", NewWarp2FromConfig) // deprecated
}

// go:generate go run ../cmd/tools/decorate.go -f decorateWarp2 -b *Warp2 -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewWarpFromConfig creates a new configurable charger
func NewWarp2FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		mqtt.Config   `mapstructure:",squash"`
		Topic         string
		EnergyManager string
		Timeout       time.Duration
	}{
		Topic:   warp.RootTopic,
		Timeout: warp.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarp2(cc.Config, cc.Topic, cc.EnergyManager, cc.Timeout)
	if err != nil {
		return nil, err
	}

	var currentPower, totalEnergy func() (float64, error)
	if wb.hasFeature(cc.Topic, warp.FeatureMeter) {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
	}

	var currents func() (float64, float64, float64, error)
	if wb.hasFeature(cc.Topic, warp.FeatureMeterPhases) {
		currents = wb.currents
	}

	var identity func() (string, error)
	if wb.hasFeature(cc.Topic, warp.FeatureNfc) {
		identity = wb.identify
	}

	var phases func(int) error
	if cc.EnergyManager != "" {
		if res, err := wb.emState(); err == nil && res.ExternalControl != 1 {
			phases = wb.phases1p3p
		}
	}

	return decorateWarp2(wb, currentPower, totalEnergy, currents, identity, phases), err
}

// NewWarp2 creates a new configurable charger
func NewWarp2(mqttconf mqtt.Config, topic, emTopic string, timeout time.Duration) (*Warp2, error) {
	log := util.NewLogger("warp")

	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	wb := &Warp2{
		log:     log,
		client:  client,
		current: 6000, // mA
	}

	// timeout handler
	to := provider.NewTimeoutHandler(provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/low_level_state", topic), timeout,
	).StringGetter())

	stringG := func(topic string) func() (string, error) {
		g := provider.NewMqtt(log, client, topic, 0).StringGetter()
		return to.StringGetter(g)
	}

	wb.maxcurrentG = stringG(fmt.Sprintf("%s/evse/external_current", topic))
	wb.statusG = stringG(fmt.Sprintf("%s/evse/state", topic))
	wb.meterG = stringG(fmt.Sprintf("%s/meter/values", topic))
	wb.meterDetailsG = stringG(fmt.Sprintf("%s/meter/all_values", topic))
	wb.chargeG = stringG(fmt.Sprintf("%s/charge_tracker/current_charge", topic))
	wb.userconfigG = stringG(fmt.Sprintf("%s/users/config", topic))

	wb.maxcurrentS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/external_current_update", topic), 0).
		WithPayload(`{ "current": ${maxcurrent} }`).
		IntSetter("maxcurrent")

	// wb.emConfigG = stringG(fmt.Sprintf("%s/energy_manager/config", emTopic))
	wb.emStateG = stringG(fmt.Sprintf("%s/energy_manager/state", emTopic))
	wb.phasesS = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/energy_manager/external_control_update", emTopic), 0).
		WithPayload(`{ "phases_wanted": ${phases} }`).
		IntSetter("phases")

	return wb, nil
}

func (wb *Warp2) hasFeature(root, feature string) bool {
	if wb.features != nil {
		return slices.Contains(wb.features, feature)
	}

	topic := fmt.Sprintf("%s/info/features", root)

	if data, err := provider.NewMqtt(wb.log, wb.client, topic, 0).StringGetter()(); err == nil {
		if err := json.Unmarshal([]byte(data), &wb.features); err == nil {
			return slices.Contains(wb.features, feature)
		}
	}

	return false
}

// Enable implements the api.Charger interface
func (wb *Warp2) Enable(enable bool) error {
	var current int64
	if enable {
		current = wb.current
	}
	return wb.maxcurrentS(current)
}

// Enabled implements the api.Charger interface
func (wb *Warp2) Enabled() (bool, error) {
	var res warp.EvseExternalCurrent

	s, err := wb.maxcurrentG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.Current >= 6000, err
}

// Status implements the api.Charger interface
func (wb *Warp2) Status() (api.ChargeStatus, error) {
	res := api.StatusNone

	s, err := wb.statusG()
	if err != nil {
		return res, err
	}

	var status warp.EvseState
	if err := json.Unmarshal([]byte(s), &status); err != nil {
		return res, err
	}

	switch status.Iec61851State {
	case 0:
		res = api.StatusA
	case 1:
		res = api.StatusB
	case 2:
		res = api.StatusC
	default:
		err = fmt.Errorf("invalid status: %d", status.Iec61851State)
	}

	return res, err
}

// MaxCurrent implements the api.Charger interface
func (wb *Warp2) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Warp2)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Warp2) MaxCurrentMillis(current float64) error {
	curr := int64(current * 1e3)
	err := wb.maxcurrentS(curr)
	if err == nil {
		wb.current = curr
	}
	return err
}

// CurrentPower implements the api.Meter interface
func (wb *Warp2) currentPower() (float64, error) {
	var res warp.MeterValues

	s, err := wb.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.Power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Warp2) totalEnergy() (float64, error) {
	var res warp.MeterValues

	s, err := wb.meterG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.EnergyAbs, err
}

// currents implements the api.MeterCurrrents interface
func (wb *Warp2) currents() (float64, float64, float64, error) {
	var res []float64

	s, err := wb.meterDetailsG()
	if err != nil {
		return 0, 0, 0, err
	}

	if err := json.Unmarshal([]byte(s), &res); err != nil {
		return 0, 0, 0, err
	}

	if len(res) <= 5 {
		return 0, 0, 0, errors.New("invalid length")
	}

	return res[3], res[4], res[5], nil
}

func (wb *Warp2) identify() (string, error) {
	var res warp.ChargeTrackerCurrentCharge

	s, err := wb.chargeG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res.AuthorizationInfo.TagId, err
}

// func (wb *Warp2) emConfig() (warp.EmConfig, error) {
// 	var res warp.EmConfig

// 	s, err := wb.emConfigG()
// 	if err == nil {
// 		err = json.Unmarshal([]byte(s), &res)
// 	}

// 	return res, err
// }

func (wb *Warp2) emState() (warp.EmState, error) {
	var res warp.EmState

	s, err := wb.emStateG()
	if err == nil {
		err = json.Unmarshal([]byte(s), &res)
	}

	return res, err
}

func (wb *Warp2) phases1p3p(phases int) error {
	res, err := wb.emState()
	if err != nil {
		return err
	}

	if res.ExternalControl > 0 {
		return fmt.Errorf("external control not available: %s", res.ExternalControl.String())
	}

	return wb.phasesS(int64(phases))
}
