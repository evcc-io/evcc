package charger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/evcc"
	"github.com/enbility/eebus-go/usecases/cem/evcem"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
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
	cem *eebus.CustomerEnergyManagement
	ev  spineapi.EntityRemoteInterface

	mux     sync.RWMutex
	log     *util.Logger
	lp      loadpoint.API
	minMaxG func() (minMax, error)

	limitUpdated time.Time // time of last limit change

	enabled   bool
	reconnect bool
	current   float64

	connector *eebus.Connector
}

func init() {
	registry.AddCtx("eebus", NewEEBusFromConfig)
}

// NewEEBusFromConfig creates an EEBus charger from generic config
func NewEEBusFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		Ski           string
		Ip            string
		Meter         bool
		ChargedEnergy *bool
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// default true
	hasChargedEnergy := cc.ChargedEnergy == nil || *cc.ChargedEnergy

	return NewEEBus(ctx, cc.Ski, cc.Ip, cc.Meter, hasChargedEnergy)
}

//go:generate go tool decorate -f decorateEEBus -b *EEBus -r api.Charger -t api.Meter,api.PhaseCurrents,api.ChargeRater

// newEEBus creates and initializes a raw *EEBus charger.
// It registers the device with the EEBus instance and waits for the connection.
func newEEBus(ctx context.Context, ski, ip string) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:     util.NewLogger("eebus"),
		current: 6,
		cem:     eebus.Instance.CustomerEnergyManagement(),
	}

	c.connector = eebus.NewConnector()
	c.minMaxG = util.Cached(c.minMax, time.Second)

	if err := eebus.Instance.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.connector.Wait(ctx); err != nil {
		eebus.Instance.UnregisterDevice(ski, c)
		return nil, err
	}

	// unregister device when context is cancelled (e.g. UI config validation)
	go func() {
		<-ctx.Done()
		eebus.Instance.UnregisterDevice(ski, c)
	}()

	return c, nil
}

// NewEEBus creates EEBus charger
func NewEEBus(ctx context.Context, ski, ip string, hasMeter, hasChargedEnergy bool) (api.Charger, error) {
	c, err := newEEBus(ctx, ski, ip)
	if err != nil {
		return nil, err
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

// Connect implements the eebus.Device interface.
// On SHIP/SPINE disconnect we drop the cached EV entity reference. EvDisconnected
// only fires on a SPINE EntityChange/Remove, not on SHIP-level disconnect, so
// without this we could keep querying an orphan entity until the next reconnect
// re-fires EvConnected.
func (c *EEBus) Connect(connected bool) {
	c.connector.Connect(connected)

	if connected {
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.ev = nil
}

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

	return c.ev, c.ev != nil && c.cem.EvCC.EVConnected(c.ev)
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
		limitsMin, _, _, err := c.cem.OpEV.CurrentLimits(evEntity)
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

	currentState, err := c.cem.EvCC.ChargeState(evEntity)
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

	limits, err := c.cem.OpEV.LoadControlLimits(evEntity)
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
		comStandard, err := c.cem.EvCC.CommunicationStandard(evEntity)
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
	if !c.cem.OpEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		return api.ErrNotAvailable
	}

	_, maxLimits, _, err := c.cem.OpEV.CurrentLimits(evEntity)
	if err != nil {
		c.log.DEBUG.Println("no limits from the EVSE are provided:", err)
	}

	// setup the obligation limit data structure
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

	// always set overload protection limits (obligation)
	if _, err := c.cem.OpEV.WriteLoadControlLimits(evEntity, limits, nil); err != nil {
		return err
	}

	// additionally set self-consumption recommendation limits if available
	c.writeOscevLimits(evEntity, current)

	c.mux.Lock()
	defer c.mux.Unlock()

	c.limitUpdated = time.Now()

	return nil
}

// writeOscevLimits writes OSCEV recommendation limits if the use case is available.
// An active recommendation triggers the EV to charge with surplus energy.
// An inactive recommendation is equivalent to no recommendation existing.
func (c *EEBus) writeOscevLimits(evEntity spineapi.EntityRemoteInterface, current float64) {
	if !c.cem.OscEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		return
	}

	// OSCEV requires recommendation limits to be available
	if _, err := c.cem.OscEV.LoadControlLimits(evEntity); err != nil {
		return
	}

	minLimits, _, _, err := c.cem.OscEV.CurrentLimits(evEntity)
	if err != nil {
		return
	}

	var limits []ucapi.LoadLimitsPhase
	for phase := range len(ucapi.PhaseNameMapping) {
		limit := ucapi.LoadLimitsPhase{
			Phase:    ucapi.PhaseNameMapping[phase],
			IsActive: false,
			Value:    current,
		}

		// below min charging current there is nothing to recommend
		// in contrast to OPEV the max value has to be active to trigger the recommendation to have any effect
		if phase < len(minLimits) {
			limit.IsActive = current >= minLimits[phase]
		}

		limits = append(limits, limit)
	}

	if _, err := c.cem.OscEV.WriteLoadControlLimits(evEntity, limits, nil); err != nil {
		c.log.DEBUG.Println("failed to write OSCEV limits:", err)
	}
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
	if c.cem.EvCem.IsScenarioAvailableAtEntity(evEntity, 2) {
		// is power data available for real? Elli Gen1 says it supports it, but doesn't provide any data
		if powerData, err := c.cem.EvCem.PowerPerPhase(evEntity); err == nil {
			powers = powerData
		}
	}

	// if no power data is available, and currents are reported to be supported, use currents
	if len(powers) == 0 && c.cem.EvCem.IsScenarioAvailableAtEntity(evEntity, 1) {
		// no power provided, calculate from current
		if currents, err := c.cem.EvCem.CurrentPerPhase(evEntity); err == nil {
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

	if !c.cem.EvCem.IsScenarioAvailableAtEntity(evEntity, 3) {
		return 0, api.ErrNotAvailable
	}

	energy, err := c.cem.EvCem.EnergyCharged(evEntity)
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
	if !c.cem.EvCem.IsScenarioAvailableAtEntity(evEntity, 1) {
		return 0, 0, 0, api.ErrNotAvailable
	}

	c.mux.Lock()
	ts := c.limitUpdated
	c.mux.Unlock()

	// if the last limit update is not zero (meaning no measurement was provided yet)
	// only consider this an error, if the last limit update is older than 15 seconds
	// this covers the case where this function may be called shortly after setting a limit
	// but too short for a measurement can even be received
	if d := time.Since(ts); d > 15*time.Second && !ts.IsZero() {
		return 0, 0, 0, api.ErrNotAvailable
	}

	res, err := c.cem.EvCem.CurrentPerPhase(evEntity)
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

	if identification, err := c.cem.EvCC.Identifications(evEntity); err == nil && len(identification) > 0 {
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

	if !c.cem.EvSoc.IsScenarioAvailableAtEntity(evEntity, 1) {
		return 0, api.ErrNotAvailable
	}

	soc, err := c.cem.EvSoc.StateOfCharge(evEntity)
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

	minLimits, maxLimits, _, err := c.cem.OpEV.CurrentLimits(evEntity)
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
