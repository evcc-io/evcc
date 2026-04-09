package charger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/features/client"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/evcc"
	"github.com/enbility/eebus-go/usecases/cem/evcem"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
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

	*eebus.Connector
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

	c.Connector = eebus.NewConnector()
	c.minMaxG = util.Cached(c.minMax, time.Second)

	if err := eebus.Instance.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.Wait(ctx); err != nil {
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

// opevFilter matches the OpEV obligation-max overload-protection limit descriptions.
// Must stay in sync with eebus-go usecases/cem/opev/public.go WriteLoadControlLimits.
var opevFilter = model.LoadControlLimitDescriptionDataType{
	LimitType:     lo.ToPtr(model.LoadControlLimitTypeTypeMaxValueLimit),
	LimitCategory: lo.ToPtr(model.LoadControlCategoryTypeObligation),
	Unit:          lo.ToPtr(model.UnitOfMeasurementTypeA),
	ScopeType:     lo.ToPtr(model.ScopeTypeTypeOverloadProtection),
}

// oscevFilter matches the OSCEV recommendation-max self-consumption limit descriptions.
// Must stay in sync with eebus-go usecases/cem/oscev/public.go WriteLoadControlLimits.
var oscevFilter = model.LoadControlLimitDescriptionDataType{
	LimitType:     lo.ToPtr(model.LoadControlLimitTypeTypeMaxValueLimit),
	LimitCategory: lo.ToPtr(model.LoadControlCategoryTypeRecommendation),
	Unit:          lo.ToPtr(model.UnitOfMeasurementTypeA),
	ScopeType:     lo.ToPtr(model.ScopeTypeTypeSelfConsumption),
}

// writeCurrentLimitData writes OpEV obligation and (if available) OSCEV
// recommendation limits to the EV in a single atomic spine-level write.
//
// Background: the currently pinned eebus-go version does not send SPINE
// partial writes — every LoadControl update has to ship the full limit list.
// If we write OpEV and OSCEV separately (as the CemOPEV/CemOSCEV use-case
// wrappers do), the second write effectively deletes the OpEV obligation
// limits the first write just installed, because both writes build their
// "full list" from the same stale feature cache. The symptom on a Porsche
// Taycan disable cycle is a persistent "charger out of sync: expected
// disabled, got enabled" warning (see log.txt on the feature/eebus-fixes
// branch for a captured trace).
//
// We cannot simply drop OSCEV: some wallbox + EV combinations will not
// charge at all unless a self-consumption recommendation is active. The fix
// is therefore to bypass the use-case wrappers and dispatch both categories
// through a single loadControl.WriteLimitData call — one merge against the
// cache, one wire message, no clobbering between categories.
func (c *EEBus) writeCurrentLimitData(evEntity spineapi.EntityRemoteInterface, current float64) error {
	// OpEV obligation-max is the required baseline for any write
	if !c.cem.OpEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		return api.ErrNotAvailable
	}

	loadControl, err := client.NewLoadControl(c.cem.LocalEntity, evEntity)
	if err != nil {
		return api.ErrNotAvailable
	}
	elConn, err := client.NewElectricalConnection(c.cem.LocalEntity, evEntity)
	if err != nil {
		return api.ErrNotAvailable
	}

	var data []model.LoadControlLimitDataType

	// OpEV: obligation max — active unless the requested current meets or
	// exceeds the phase max (in which case "no obligation" = unlimited).
	_, maxLimits, _, cerr := c.cem.OpEV.CurrentLimits(evEntity)
	if cerr != nil {
		c.log.DEBUG.Println("no limits from the EVSE are provided:", cerr)
	}
	if entries := buildPhaseLimitData(loadControl, elConn, opevFilter, computeOpevLimits(current, maxLimits)); entries != nil {
		data = append(data, entries...)
	}

	// OSCEV: recommendation max — optional. Active only when the requested
	// current is at least the phase min (a recommendation below min cannot
	// drive charging and is equivalent to no recommendation).
	if c.cem.OscEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		if _, lerr := c.cem.OscEV.LoadControlLimits(evEntity); lerr == nil {
			if minLimits, _, _, merr := c.cem.OscEV.CurrentLimits(evEntity); merr == nil {
				if entries := buildPhaseLimitData(loadControl, elConn, oscevFilter, computeOscevLimits(current, minLimits)); entries != nil {
					data = append(data, entries...)
				}
			}
		}
	}

	if len(data) == 0 {
		return api.ErrNotAvailable
	}

	if _, err := loadControl.WriteLimitData(data, nil, nil); err != nil {
		return err
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	c.limitUpdated = time.Now()
	return nil
}

// computeOpevLimits returns the per-phase OpEV obligation-max tuples for the
// given current. IsActive is false when the current equals or exceeds the
// phase max (no obligation = unlimited per EEBus semantics).
func computeOpevLimits(current float64, maxLimits []float64) []ucapi.LoadLimitsPhase {
	limits := make([]ucapi.LoadLimitsPhase, 0, len(ucapi.PhaseNameMapping))
	for phase := range len(ucapi.PhaseNameMapping) {
		limit := ucapi.LoadLimitsPhase{
			Phase:    ucapi.PhaseNameMapping[phase],
			IsActive: true,
			Value:    current,
		}
		if phase < len(maxLimits) && current >= maxLimits[phase] {
			limit.IsActive = false
		}
		limits = append(limits, limit)
	}
	return limits
}

// computeOscevLimits returns the per-phase OSCEV recommendation-max tuples for
// the given current. Unlike OpEV, the recommendation is active only when the
// current is at least the phase min — below min there is nothing useful to
// recommend and the limit must be inactive to be ignored by the EV.
func computeOscevLimits(current float64, minLimits []float64) []ucapi.LoadLimitsPhase {
	limits := make([]ucapi.LoadLimitsPhase, 0, len(ucapi.PhaseNameMapping))
	for phase := range len(ucapi.PhaseNameMapping) {
		limit := ucapi.LoadLimitsPhase{
			Phase:    ucapi.PhaseNameMapping[phase],
			IsActive: false,
			Value:    current,
		}
		if phase < len(minLimits) {
			limit.IsActive = current >= minLimits[phase]
		}
		limits = append(limits, limit)
	}
	return limits
}

// buildPhaseLimitData converts per-phase LoadLimitsPhase tuples into spine
// LoadControlLimitDataType entries for a given description filter. It mirrors
// the per-phase mapping inside eebus-go usecases/internal/loadcontrol.go
// WriteLoadControlPhaseLimits but does not dispatch a write — callers may
// accumulate entries across multiple filters and issue a single
// loadControl.WriteLimitData for atomicity.
//
// Returns nil if the filter does not match any limit description on the
// remote (e.g. the use case is not supported); individual phases are skipped
// if their specific description or parameter data is missing.
func buildPhaseLimitData(
	loadControl eebusapi.LoadControlCommonInterface,
	elConn eebusapi.ElectricalConnectionCommonInterface,
	filter model.LoadControlLimitDescriptionDataType,
	limits []ucapi.LoadLimitsPhase,
) []model.LoadControlLimitDataType {
	limitDescriptions, err := loadControl.GetLimitDescriptionsForFilter(filter)
	if err != nil || len(limitDescriptions) == 0 {
		return nil
	}

	var data []model.LoadControlLimitDataType
	for _, phaseLimit := range limits {
		paramFilter := model.ElectricalConnectionParameterDescriptionDataType{
			AcMeasuredPhases: lo.ToPtr(phaseLimit.Phase),
		}
		elParamDesc, err := elConn.GetParameterDescriptionsForFilter(paramFilter)
		if err != nil || len(elParamDesc) == 0 || elParamDesc[0].MeasurementId == nil {
			continue
		}

		var limitDesc *model.LoadControlLimitDescriptionDataType
		for _, desc := range limitDescriptions {
			if desc.MeasurementId != nil && *desc.MeasurementId == *elParamDesc[0].MeasurementId {
				safe := desc
				limitDesc = &safe
				break
			}
		}
		if limitDesc == nil || limitDesc.LimitId == nil {
			continue
		}

		limitIdData, err := loadControl.GetLimitDataForId(*limitDesc.LimitId)
		if err != nil {
			continue
		}
		if limitIdData.IsLimitChangeable != nil && !*limitIdData.IsLimitChangeable {
			continue
		}

		value := elConn.AdjustValueToBeWithinPermittedValuesForParameterId(
			phaseLimit.Value, *elParamDesc[0].ParameterId)

		data = append(data, model.LoadControlLimitDataType{
			LimitId:       limitDesc.LimitId,
			IsLimitActive: lo.ToPtr(phaseLimit.IsActive),
			Value:         model.NewScaledNumberType(value),
		})
	}
	return data
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
