package charger

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/evcc"
	"github.com/enbility/eebus-go/usecases/cem/evcem"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
)

const (
	idleFactor         = 0.6
	voltage    float64 = 230
)

type minMax struct {
	min, max float64
}

type EEBus struct {
	uc *eebus.UseCasesEVSE
	ev spineapi.EntityRemoteInterface

	mux     sync.RWMutex
	log     *util.Logger
	lp      loadpoint.API
	minMaxG func() (minMax, error)

	limitUpdated time.Time // time of last limit change

	vasVW     bool // wether the EVSE supports VW VAS with ISO15118-2
	enabled   bool
	reconnect bool
	current   float64

	*eebus.Connector
}

func init() {
	registry.Add("eebus", NewEEBusFromConfig)
}

// NewEEBusFromConfig creates an EEBus charger from generic config
func NewEEBusFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Ski           string
		Ip            string
		Meter         bool
		ChargedEnergy bool
		VasVW         bool
	}{
		ChargedEnergy: true,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(cc.Ski, cc.Ip, cc.Meter, cc.ChargedEnergy, cc.VasVW)
}

//go:generate decorate -f decorateEEBus -b *EEBus -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)"

// NewEEBus creates EEBus charger
func NewEEBus(ski, ip string, hasMeter, hasChargedEnergy, vasVW bool) (api.Charger, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:     util.NewLogger("eebus"),
		current: 6,
		vasVW:   vasVW,
		uc:      eebus.Instance.Evse(),
	}

	c.Connector = eebus.NewConnector()
	c.minMaxG = provider.Cached(c.minMax, time.Second)

	if err := eebus.Instance.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.Wait(90 * time.Second); err != nil {
		return c, err
	}

	if hasMeter {
		var energyG func() (float64, error)
		if hasChargedEnergy {
			energyG = c.chargedEnergy
		}
		return decorateEEBus(c, c.currentPower, c.currents, energyG), nil
	}

	return c, nil
}

var _ eebus.Device = (*EEBus)(nil)

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	c.mux.Lock()
	defer c.mux.Unlock()

	// EV
	switch event {
	case evcc.EvConnected:
		c.ev = entity
		c.reconnect = true

	case evcc.EvDisconnected:
		c.ev = nil

	case evcem.DataUpdateCurrentPerPhase:
		// acknowledge limit change
		c.limitUpdated = time.Time{}
	}
}

func (c *EEBus) isEvConnected() (spineapi.EntityRemoteInterface, bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.ev, c.ev != nil && c.uc.EvCC.EVConnected(c.ev)
}

// we assume that if any phase current value is > idleFactor * min Current, then charging is active and enabled is true
func (c *EEBus) isCharging(evEntity spineapi.EntityRemoteInterface) bool {
	// check if an external physical meter is assigned
	// we only want this for configured meters and not for internal meters!
	// right now it works as expected
	var minPower float64
	if c.lp != nil {
		minPower = c.lp.EffectiveMinPower()

		if c.lp.HasChargeMeter() {
			return c.lp.GetChargePower() > minPower*idleFactor
		}
	}

	// The above doesn't (yet) work for built in meters, so check the EEBUS measurements also

	// use power data if available, otherwise the method will calculate the power from the current data
	power, err := c.currentPower()
	if err != nil {
		return false
	}

	if c.lp == nil {
		limitsMin, _, _, err := c.uc.OpEV.CurrentLimits(evEntity)
		if err != nil || len(limitsMin) == 0 {
			// sometimes a min limit is not provided by the EVSE, and we can't take it from the loadpoint
			return false
		}
		minPower = limitsMin[0] * voltage
	}

	return power > minPower*idleFactor
}

// Status implements the api.Charger interface
func (c *EEBus) Status() (res api.ChargeStatus, err error) {
	evEntity, ok := c.isEvConnected()
	if !ok {
		return api.StatusA, nil
	}

	// re-set current limit after reconnect
	defer func() {
		if err != nil {
			return
		}

		c.mux.Lock()
		if !c.reconnect && (res == api.StatusB || res == api.StatusC) {
			c.mux.Unlock()
			return
		}

		c.reconnect = false
		c.mux.Unlock()

		var current float64
		if c.enabled {
			current = c.current
		}

		err = c.writeCurrentLimitData(evEntity, current)
	}()

	currentState, err := c.uc.EvCC.ChargeState(evEntity)
	if err != nil {
		return api.StatusA, nil
	}

	switch currentState {
	case ucapi.EVChargeStateTypeUnknown, ucapi.EVChargeStateTypeUnplugged: // Unplugged
		return api.StatusA, nil
	case ucapi.EVChargeStateTypeFinished, ucapi.EVChargeStateTypePaused: // Finished, Paused
		return api.StatusB, nil
	case ucapi.EVChargeStateTypeActive: // Active
		if c.isCharging(evEntity) {
			return api.StatusC, nil
		}
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", currentState)
	}
}

// Enabled implements the api.Charger interface
// should return true if the charger allows the EV to draw power
func (c *EEBus) Enabled() (bool, error) {
	// when unplugged there is no overload limit data available
	evEntity, ok := c.isEvConnected()
	if !ok {
		return c.enabled, nil
	}

	// if the VW VAS PV mode is active, use PV limits
	if c.hasActiveVASVW(evEntity) {
		limits, err := c.uc.OscEV.LoadControlLimits(evEntity)
		if err != nil {
			// there are no limits available, e.g. because the data was not received yet
			return c.enabled, nil
		}

		for _, limit := range limits {
			// check if there is an active limit set
			if limit.IsActive && limit.Value >= 1 {
				return true, nil
			}
		}

		return false, nil
	}

	limits, err := c.uc.OpEV.LoadControlLimits(evEntity)
	if err != nil {
		// there are no limits available, e.g. because the data was not received yet
		return c.enabled, nil
	}

	for _, limit := range limits {
		// for IEC61851 the pause limit is 0A, for ISO15118-2 it is 0.1A
		// instead of checking for the actual data, hardcode this, so we might run into less
		// timing issues as the data might not be received yet
		// if the limit is not active, then the maximum possible current is permitted
		if limit.IsActive && limit.Value >= 1 || !limit.IsActive {
			return true, nil
		}
	}

	return false, nil
}

// Enable implements the api.Charger interface
func (c *EEBus) Enable(enable bool) error {
	// if the ev is unplugged or the state is unknown, there is nothing to be done
	evEntity, ok := c.isEvConnected()
	if !ok {
		c.enabled = enable
		return nil
	}

	// if we disable charging with a potential but not yet known communication standard ISO15118
	// this would set allowed A value to be 0. And this would trigger ISO connections to switch to IEC!
	if !enable {
		comStandard, err := c.uc.EvCC.CommunicationStandard(evEntity)
		if err != nil || comStandard == evcc.EVCCCommunicationStandardUnknown {
			return api.ErrMustRetry
		}
	}

	var current float64
	if enable {
		current = c.current
	}

	err := c.writeCurrentLimitData(evEntity, current)
	if err == nil {
		c.enabled = enable
	}

	return err
}

// send current charging power limits to the EV
func (c *EEBus) writeCurrentLimitData(evEntity spineapi.EntityRemoteInterface, current float64) error {
	// check if the EVSE supports overload protection limits
	if !c.uc.OpEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		return api.ErrNotAvailable
	}

	_, maxLimits, _, err := c.uc.OpEV.CurrentLimits(evEntity)
	if err != nil {
		c.log.DEBUG.Println("no limits from the EVSE are provided:", err)
	}

	// setup the limit data structure
	var limits []ucapi.LoadLimitsPhase
	for phase := range len(ucapi.PhaseNameMapping) {
		limit := ucapi.LoadLimitsPhase{
			Phase:    ucapi.PhaseNameMapping[phase],
			IsActive: true,
			Value:    current,
		}

		// if the limit equals to the max allowed, then the obligation limit is actually inactive
		if phase < len(maxLimits) && current >= maxLimits[phase] {
			limit.IsActive = false
		}

		limits = append(limits, limit)
	}

	// if VAS VW is available, limits are completely covered by it
	// this way evcc can fully control the charging behavior
	if c.writeLoadControlLimitsVASVW(evEntity, limits) {
		c.mux.Lock()
		defer c.mux.Unlock()

		c.limitUpdated = time.Now()
		return nil
	}

	// make sure the recommendations are inactive, otherwise the EV won't go to sleep
	// but only if it supports OSCEV and has required data!
	if c.uc.OscEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		if _, err := c.uc.OscEV.LoadControlLimits(evEntity); err == nil {
			if err := c.disableLimits(evEntity, c.uc.OscEV); err != nil {
				return err
			}
		}
	}

	// set overload protection limits
	_, err = c.uc.OpEV.WriteLoadControlLimits(evEntity, limits, nil)
	if err == nil {
		c.mux.Lock()
		defer c.mux.Unlock()

		c.limitUpdated = time.Now()
	}

	return err
}

// returns if the connected EV has an active VW PV mode
// in this mode, the EV does not have an active charging demand
func (c *EEBus) hasActiveVASVW(evEntity spineapi.EntityRemoteInterface) bool {
	// EVSE has to support VW VAS
	if !c.vasVW {
		return false
	}

	// ISO15118-2 has to be used between EVSE and EV
	if comStandard, err := c.uc.EvCC.CommunicationStandard(evEntity); err != nil || comStandard != model.DeviceConfigurationKeyValueStringTypeISO151182ED2 {
		return false
	}

	// SoC has to be available, otherwise it is plain ISO15118-2
	// SoC has to be >= 25%, because the Taycan can't be setup with a Min SoC below 25%, oherwise obligations have to be used
	if soc, err := c.Soc(); err != nil || soc < 25 {
		return false
	}

	// Optimization of self consumption use case support has to be available
	if !c.uc.EvSoc.IsScenarioAvailableAtEntity(evEntity, 1) {
		return false
	}

	// the use case has to be reported as active
	// only then the EV has no active charging demand and will charge based on OSCEV recommendations
	// this is a workaround for EVSE changing isActive to false, even though they should
	// not announce the use case at all in that case
	for _, uci := range evEntity.Device().UseCases() {
		// check if the referenced entity address is identical to the ev entity address
		// the address may not exist, as it only available since SPINE 1.3
		if uci.Address != nil &&
			evEntity.Address() != nil &&
			slices.Compare(uci.Address.Entity, evEntity.Address().Entity) != 0 {
			continue
		}

		for _, uc := range uci.UseCaseSupport {
			if uc.UseCaseName != nil && *uc.UseCaseName == model.UseCaseNameTypeOptimizationOfSelfConsumptionDuringEVCharging &&
				uc.UseCaseAvailable != nil && *uc.UseCaseAvailable {
				return true
			}
		}
	}

	return false
}

// provides support for the special VW VAS ISO15118-2 charging behavior if supported
// will return false if it isn't supported or successful
//
// this functionality allows to fully control charging without the EV actually having a
// charging demand by itself
func (c *EEBus) writeLoadControlLimitsVASVW(evEntity spineapi.EntityRemoteInterface, limits []ucapi.LoadLimitsPhase) bool {
	if !c.hasActiveVASVW(evEntity) {
		return false
	}

	// check if the EVSE supports optimization of self consumption limits
	if !c.uc.OscEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		return false
	}

	// OSCEV requires recommendation limits to be available
	if _, err := c.uc.OscEV.LoadControlLimits(evEntity); err != nil {
		return false
	}

	// on OSCEV all limits have to be active except they are set to the default value
	minLimits, _, _, err := c.uc.OscEV.CurrentLimits(evEntity)
	if err != nil {
		return false
	}

	for index, item := range limits {
		// if the limit is equal or bigger than the min allowed, then the recommendation limit is active, otherwise it is not
		limits[index].IsActive = false
		if index < len(minLimits) {
			limits[index].IsActive = item.Value >= minLimits[index]
		}
	}

	// set recommendation limits
	if _, err := c.uc.OscEV.WriteLoadControlLimits(evEntity, limits, nil); err != nil {
		return false
	}

	if err := c.disableLimits(evEntity, c.uc.OpEV); err != nil {
		return false
	}

	return true
}

type eebusLimitController interface {
	LoadControlLimits(spineapi.EntityRemoteInterface) ([]ucapi.LoadLimitsPhase, error)
	WriteLoadControlLimits(spineapi.EntityRemoteInterface, []ucapi.LoadLimitsPhase, func(result model.ResultDataType)) (*model.MsgCounterType, error)
}

// make sure the limits are inactive, otherwise the EV won't go to sleep
func (c *EEBus) disableLimits(evEntity spineapi.EntityRemoteInterface, uc eebusLimitController) error {
	limits, err := uc.LoadControlLimits(evEntity)
	if err != nil {
		return err
	}

	var writeNeeded bool
	for index, item := range limits {
		if item.IsActive {
			limits[index].IsActive = false
			writeNeeded = true
		}
	}

	if writeNeeded {
		_, err = uc.WriteLoadControlLimits(evEntity, limits, nil)
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *EEBus) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EEBus)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *EEBus) MaxCurrentMillis(current float64) error {
	evEntity, ok := c.isEvConnected()
	if !ok {
		c.current = current
		return nil
	}

	err := c.writeCurrentLimitData(evEntity, current)
	if err == nil {
		c.current = current
	}

	return nil
}

// CurrentPower implements the api.Meter interface
func (c *EEBus) currentPower() (float64, error) {
	evEntity, ok := c.isEvConnected()
	if !ok {
		return 0, nil
	}

	// does the EVSE provide power data?
	var powers []float64
	if c.uc.EvCem.IsScenarioAvailableAtEntity(evEntity, 2) {
		// is power data available for real? Elli Gen1 says it supports it, but doesn't provide any data
		if powerData, err := c.uc.EvCem.PowerPerPhase(evEntity); err == nil {
			powers = powerData
		}
	}

	// if no power data is available, and currents are reported to be supported, use currents
	if len(powers) == 0 && c.uc.EvCem.IsScenarioAvailableAtEntity(evEntity, 1) {
		// no power provided, calculate from current
		if currents, err := c.uc.EvCem.CurrentPerPhase(evEntity); err == nil {
			for _, current := range currents {
				powers = append(powers, current*voltage)
			}
		}
	}

	// if still no power data is available, return an error
	if len(powers) == 0 {
		return 0, api.ErrNotAvailable
	}

	return lo.Sum(powers), nil
}

// ChargedEnergy implements the api.ChargeRater interface
func (c *EEBus) chargedEnergy() (float64, error) {
	evEntity, ok := c.isEvConnected()
	if !ok {
		return 0, nil
	}

	if !c.uc.EvCem.IsScenarioAvailableAtEntity(evEntity, 3) {
		return 0, api.ErrNotAvailable
	}

	energy, err := c.uc.EvCem.EnergyCharged(evEntity)
	if err != nil {
		return 0, api.ErrNotAvailable
	}

	return energy / 1e3, nil
}

// Currents implements the api.PhaseCurrents interface
func (c *EEBus) currents() (float64, float64, float64, error) {
	evEntity, ok := c.isEvConnected()
	if !ok {
		return 0, 0, 0, nil
	}

	// check if the EVSE supports currents
	if !c.uc.EvCem.IsScenarioAvailableAtEntity(evEntity, 1) {
		return 0, 0, 0, api.ErrNotAvailable
	}

	c.mux.Lock()
	ts := c.limitUpdated
	c.mux.Unlock()

	// if the last limit update is not zero (meaning no measurement was provided yet)
	// only consider this an error, if the last limit update is older than 15 seconds
	// this covers the case where this function may be called shortly after setting a limit
	// but too short for a measurement can even be received
	if d := time.Now().Sub(ts); d > 15*time.Second && !ts.IsZero() {
		return 0, 0, 0, api.ErrNotAvailable
	}

	res, err := c.uc.EvCem.CurrentPerPhase(evEntity)
	if err != nil {
		return 0, 0, 0, eebus.WrapError(err)
	}

	// fill phases
	for len(res) < 3 {
		res = append(res, 0)
	}

	return res[0], res[1], res[2], nil
}

var _ api.Identifier = (*EEBus)(nil)

// Identify implements the api.Identifier interface
func (c *EEBus) Identify() (string, error) {
	evEntity, ok := c.isEvConnected()
	if !ok {
		return "", nil
	}

	if identification, err := c.uc.EvCC.Identifications(evEntity); err == nil && len(identification) > 0 {
		// return the first identification for now
		// later this could be multiple, e.g. MAC Address and PCID
		return identification[0].Value, nil
	}

	return "", nil
}

var _ api.Battery = (*EEBus)(nil)

// Soc implements the api.Battery interface
func (c *EEBus) Soc() (float64, error) {
	evEntity, ok := c.isEvConnected()
	if !ok {
		return 0, api.ErrNotAvailable
	}

	if !c.uc.EvSoc.IsScenarioAvailableAtEntity(evEntity, 1) {
		return 0, api.ErrNotAvailable
	}

	soc, err := c.uc.EvSoc.StateOfCharge(evEntity)
	if err != nil {
		return 0, api.ErrNotAvailable
	}

	return soc, nil
}

var _ api.CurrentLimiter = (*EEBus)(nil)

func (c *EEBus) minMax() (minMax, error) {
	var zero minMax

	evEntity, ok := c.isEvConnected()
	if !ok {
		return zero, nil
	}

	minLimits, maxLimits, _, err := c.uc.OpEV.CurrentLimits(evEntity)
	if err != nil {
		return zero, eebus.WrapError(err)
	}

	if len(minLimits) == 0 || len(maxLimits) == 0 {
		return zero, api.ErrNotAvailable
	}

	return minMax{minLimits[0], maxLimits[0]}, nil
}

func (c *EEBus) GetMinMaxCurrent() (float64, float64, error) {
	minMax, err := c.minMaxG()
	return minMax.min, minMax.max, err
}

var _ loadpoint.Controller = (*EEBus)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *EEBus) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
