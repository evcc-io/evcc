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
	maxcurrentG   func(any) error
	statusG       func(any) error
	meterG        func(any) error
	meterDetailsG func(any) error
	chargeG       func(any) error
	emStateG      func(any) error
	emLowLevelG   func(any) error
	maxcurrentS   func(int64) error
	phasesS       func(int64) error
	current       int64
}

func init() {
	registry.Add("warp2", NewWarp2FromConfig)
	registry.Add("warp-fw2", NewWarp2FromConfig) // deprecated
}

//go:generate go run ../cmd/tools/decorate.go -f decorateWarp2 -b *Warp2 -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

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
	if wb.hasFeature(cc.Topic, warp.FeatureMeter, cc.Timeout) {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
	}

	var currents, voltages func() (float64, float64, float64, error)
	if wb.hasFeature(cc.Topic, warp.FeatureMeterPhases, cc.Timeout) {
		currents = wb.currents
		voltages = wb.voltages
	}

	var identity func() (string, error)
	if wb.hasFeature(cc.Topic, warp.FeatureNfc, cc.Timeout) {
		identity = wb.identify
	}

	var phases func(int) error
	var getPhases func() (int, error)
	if cc.EnergyManager != "" {
		if res, err := wb.emState(); err == nil && res.ExternalControl != 1 {
			phases = wb.phases1p3p
			getPhases = wb.getPhases
		}
	}

	return decorateWarp2(wb, currentPower, totalEnergy, currents, voltages, identity, phases, getPhases), err
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
	h, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/evse/low_level_state", topic), timeout).StringGetter()
	if err != nil {
		return nil, err
	}
	to := provider.NewTimeoutHandler(h)

	mq := func(s string, args ...any) *provider.Mqtt {
		return provider.NewMqtt(log, client, fmt.Sprintf(s, args...), 0)
	}

	wb.maxcurrentG, err = to.JsonGetter(mq("%s/evse/external_current", topic))
	if err != nil {
		return nil, err
	}
	wb.statusG, err = to.JsonGetter(mq("%s/evse/state", topic))
	if err != nil {
		return nil, err
	}
	wb.meterG, err = to.JsonGetter(mq("%s/meter/values", topic))
	if err != nil {
		return nil, err
	}
	wb.meterDetailsG, err = to.JsonGetter(mq("%s/meter/all_values", topic))
	if err != nil {
		return nil, err
	}
	wb.chargeG, err = to.JsonGetter(mq("%s/charge_tracker/current_charge", topic))
	if err != nil {
		return nil, err
	}

	wb.maxcurrentS, err = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/evse/external_current_update", topic), 0).
		WithPayload(`{ "current": ${maxcurrent} }`).
		IntSetter("maxcurrent")
	if err != nil {
		return nil, err
	}

	wb.emStateG, err = to.JsonGetter(mq("%s/power_manager/state", emTopic))
	if err != nil {
		return nil, err
	}

	wb.phasesS, err = provider.NewMqtt(log, client,
		fmt.Sprintf("%s/power_manager/external_control_update", emTopic), 0).
		WithPayload(`{ "phases_wanted": ${phases} }`).
		IntSetter("phases")
	if err != nil {
		return nil, err
	}

	wb.emLowLevelG, err = to.JsonGetter(mq("%s/power_manager/low_level_state", emTopic))
	if err != nil {
		return nil, err
	}

	return wb, nil
}

func (wb *Warp2) hasFeature(root, feature string, timeout time.Duration) bool {
	if wb.features != nil {
		return slices.Contains(wb.features, feature)
	}

	topic := fmt.Sprintf("%s/info/features", root)

	if dataG, err := provider.NewMqtt(wb.log, wb.client, topic, timeout).StringGetter(); err == nil {
		if data, err := dataG(); err == nil {
			if err := json.Unmarshal([]byte(data), &wb.features); err == nil {
				return slices.Contains(wb.features, feature)
			}
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
	err := wb.maxcurrentG(&res)
	return res.Current >= 6000, err
}

// Status implements the api.Charger interface
func (wb *Warp2) Status() (api.ChargeStatus, error) {
	res := api.StatusNone

	var status warp.EvseState
	err := wb.statusG(&status)
	if err != nil {
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
	err := wb.meterG(&res)
	return res.Power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Warp2) totalEnergy() (float64, error) {
	var res warp.MeterValues
	err := wb.meterG(&res)
	return res.EnergyAbs, err
}

func (wb *Warp2) meterValues() ([]float64, error) {
	var res []float64
	err := wb.meterDetailsG(&res)

	if err == nil && len(res) <= 5 {
		err = errors.New("invalid length")
	}

	return res, err
}

// currents implements the api.MeterCurrrents interface
func (wb *Warp2) currents() (float64, float64, float64, error) {
	res, err := wb.meterValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return res[3], res[4], res[5], nil
}

// voltages implements the api.MeterVoltages interface
func (wb *Warp2) voltages() (float64, float64, float64, error) {
	res, err := wb.meterValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return res[0], res[1], res[2], nil
}

func (wb *Warp2) identify() (string, error) {
	var res warp.ChargeTrackerCurrentCharge
	err := wb.chargeG(&res)
	return res.AuthorizationInfo.TagId, err
}

func (wb *Warp2) emState() (warp.EmState, error) {
	var res warp.EmState
	err := wb.emStateG(&res)
	return res, err
}

func (wb *Warp2) emLowLevelState() (warp.EmLowLevelState, error) {
	var res warp.EmLowLevelState
	err := wb.emLowLevelG(&res)
	return res, err
}

// phases1p3p implements the api.PhaseSwitcher interface
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

// getPhases implements the api.PhaseGetter interface
func (wb *Warp2) getPhases() (int, error) {
	res, err := wb.emLowLevelState()
	if err != nil {
		return 0, err
	}

	if res.Is3phase {
		return 3, nil
	}

	return 1, nil
}
