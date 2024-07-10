package charger

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/evcc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/eebus"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

const (
	maxIdRequestTimespan         = time.Second * 120
	idleFactor                   = 0.6
	voltage              float64 = 230
)

type minMax struct {
	min, max float64
}

type EEBus struct {
	ski string

	uc *eebus.UseCasesEVSE
	ev spineapi.EntityRemoteInterface

	log     *util.Logger
	lp      loadpoint.API
	minMaxG func() (minMax, error)

	communicationStandard model.DeviceConfigurationKeyValueStringType
	vasVW                 bool // wether the EVSE supports VW VAS with ISO15118-2

	expectedEnableUnpluggedState bool
	current                      float64

	currentLimit float64

	lastIsChargingCheck  time.Time
	lastIsChargingResult bool

	connected     bool
	connectedC    chan bool
	connectedTime time.Time

	muxEntity sync.Mutex
	mux       sync.Mutex
}

func init() {
	registry.Add("eebus", NewEEBusFromConfig)
}

// NewEEBusFromConfig creates an EEBus charger from generic config
func NewEEBusFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Ski           string
		Ip_           string `mapstructure:"ip"` // deprecated
		Meter         bool
		ChargedEnergy bool
		VasVW         bool
	}{
		ChargedEnergy: true,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(cc.Ski, cc.Meter, cc.ChargedEnergy, cc.VasVW)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateEEBus -b *EEBus -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)"

// NewEEBus creates EEBus charger
func NewEEBus(ski string, hasMeter, hasChargedEnergy, vasVW bool) (api.Charger, error) {
	log := util.NewLogger("eebus")

	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		ski:        ski,
		log:        log,
		connectedC: make(chan bool, 1),
		current:    6,
		vasVW:      vasVW,
	}

	c.uc = eebus.Instance.RegisterEVSE(ski, c)

	c.minMaxG = provider.Cached(c.minMax, time.Second)

	if err := c.waitForConnection(); err != nil {
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

// waitForConnection wait for initial connection and returns an error on failure
func (c *EEBus) waitForConnection() error {
	timeout := time.After(90 * time.Second)
	for {
		select {
		case <-timeout:
			return os.ErrDeadlineExceeded
		case connected := <-c.connectedC:
			if connected {
				return nil
			}
		}
	}
}

func (c *EEBus) setEvEntity(entity spineapi.EntityRemoteInterface) {
	c.muxEntity.Lock()
	defer c.muxEntity.Unlock()

	c.ev = entity
}

func (c *EEBus) evEntity() spineapi.EntityRemoteInterface {
	c.muxEntity.Lock()
	defer c.muxEntity.Unlock()

	return c.ev
}

// EEBUSDeviceInterface

func (c *EEBus) DeviceConnect() {
	c.log.TRACE.Println("connect ski:", c.ski)

	c.expectedEnableUnpluggedState = false
	c.setDefaultValues()
	c.setConnected(true)
}

func (c *EEBus) DeviceDisconnect() {
	c.log.TRACE.Println("disconnect ski:", c.ski)

	c.expectedEnableUnpluggedState = false
	c.setConnected(false)
	c.setDefaultValues()
}

// UseCase specific events
func (c *EEBus) UseCaseEventCB(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	switch event {
	// EV
	case evcc.EvConnected:
		c.log.TRACE.Println("EV Connected")
		c.setEvEntity(entity)
		c.currentLimit = -1
	case evcc.EvDisconnected:
		c.log.TRACE.Println("EV Disconnected")
		c.setEvEntity(nil)
		c.currentLimit = -1
	}
}

func (c *EEBus) setDefaultValues() {
	c.communicationStandard = evcc.EVCCCommunicationStandardUnknown
	c.lastIsChargingCheck = time.Now().Add(-time.Hour * 1)
	c.lastIsChargingResult = false
}

// set wether the EVSE is connected
func (c *EEBus) setConnected(connected bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if connected && !c.connected {
		c.connectedTime = time.Now()
	}

	select {
	case c.connectedC <- connected:
	default:
	}

	c.connected = connected
}

func (c *EEBus) isConnected() bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.connected
}

var _ api.CurrentLimiter = (*EEBus)(nil)

func (c *EEBus) minMax() (minMax, error) {
	evEntity := c.evEntity()
	if !c.uc.EvCC.EVConnected(evEntity) {
		return minMax{}, errors.New("no ev connected")
	}

	minLimits, maxLimits, _, err := c.uc.OpEV.CurrentLimits(evEntity)
	if err != nil {
		if err == eebusapi.ErrDataNotAvailable {
			err = api.ErrNotAvailable
		}
		return minMax{}, err
	}

	if len(minLimits) == 0 || len(maxLimits) == 0 {
		return minMax{}, api.ErrNotAvailable
	}

	return minMax{minLimits[0], maxLimits[0]}, nil
}

func (c *EEBus) GetMinMaxCurrent() (float64, float64, error) {
	minMax, err := c.minMaxG()
	return minMax.min, minMax.max, err
}

// we assume that if any phase current value is > idleFactor * min Current, then charging is active and enabled is true
func (c *EEBus) isCharging() bool {
	evEntity := c.evEntity()
	if !c.uc.EvCC.EVConnected(evEntity) {
		return false
	}

	// check if an external physical meter is assigned
	// we only want this for configured meters and not for internal meters!
	// right now it works as expected
	if c.lp != nil && c.lp.HasChargeMeter() {
		// we only check ever 10 seconds, maybe we can use the config interval duration
		if time.Since(c.lastIsChargingCheck) >= 10*time.Second {
			c.lastIsChargingCheck = time.Now()
			c.lastIsChargingResult = false
			// compare charge power for all phases to 0.6 * min. charge power of a single phase
			if c.lp.GetChargePower() > c.lp.EffectiveMinPower()*idleFactor {
				c.lastIsChargingResult = true
				return true
			}
		} else if c.lastIsChargingResult {
			return true
		}
	}

	// The above doesn't (yet) work for built in meters, so check the EEBUS measurements also
	currents, err := c.uc.EvCem.CurrentPerPhase(evEntity)
	if err != nil {
		return false
	}
	limitsMin, _, _, err := c.uc.OpEV.CurrentLimits(evEntity)
	if err != nil || limitsMin == nil || len(limitsMin) == 0 {
		return false
	}

	var phasesCurrent float64
	for _, phaseCurrent := range currents {
		phasesCurrent += phaseCurrent
	}

	// require sum of all phase currents to be > 0.6 * a single phase minimum
	// in some scenarios, e.g. Cayenne Hybrid, sometimes the meter of a PMCC device
	// reported 600W, even tough the car was not charging
	limitMin := limitsMin[0]
	return phasesCurrent > limitMin*idleFactor
}

// Status implements the api.Charger interface
func (c *EEBus) Status() (api.ChargeStatus, error) {
	evEntity := c.evEntity()
	if !c.isConnected() {
		return api.StatusNone, api.ErrTimeout
	}

	if !c.uc.EvCC.EVConnected(evEntity) {
		c.expectedEnableUnpluggedState = false
		return api.StatusA, nil
	}

	currentState, err := c.uc.EvCC.ChargeState(evEntity)
	if err != nil {
		return api.StatusNone, err
	}

	switch currentState {
	case ucapi.EVChargeStateTypeUnknown, ucapi.EVChargeStateTypeUnplugged: // Unplugged
		c.expectedEnableUnpluggedState = false
		return api.StatusA, nil
	case ucapi.EVChargeStateTypeFinished, ucapi.EVChargeStateTypePaused: // Finished, Paused
		return api.StatusB, nil
	case ucapi.EVChargeStateTypeActive: // Active
		if c.isCharging() {
			return api.StatusC, nil
		}
		return api.StatusB, nil
	case ucapi.EVChargeStateTypeError: // Error
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("%s properties unknown result: %s", c.ski, currentState)
	}
}

// Enabled implements the api.Charger interface
// should return true if the charger allows the EV to draw power
func (c *EEBus) Enabled() (bool, error) {
	// when unplugged there is no overload limit data available
	evEntity := c.evEntity()
	state, err := c.Status()
	if err != nil || state == api.StatusA || evEntity == nil {
		return c.expectedEnableUnpluggedState, nil
	}

	// if the EV is charging
	if state == api.StatusC {
		return true, nil
	}

	// if the VW VAS PV mode is active, use PV limits
	if c.hasActiveVASVW() {
		limits, err := c.uc.OscEV.LoadControlLimits(evEntity)
		if err != nil {
			// there are no limits available, e.g. because the data was not received yet
			return true, nil
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
		// there are limits available, e.g. because the data was not received yet
		return true, nil
	}

	for _, limit := range limits {
		// for IEC61851 the pause limit is 0A, for ISO15118-2 it is 0.1A
		// instead of checking for the actual data, hardcode this, so we might run into less
		// timing issues as the data might not be received yet
		// if the limit is not active, then the maximum possible current is permitted
		if (limit.IsActive && limit.Value >= 1) ||
			!limit.IsActive {
			return true, nil
		}
	}

	return false, nil
}

// Enable implements the api.Charger interface
func (c *EEBus) Enable(enable bool) error {
	// if the ev is unplugged or the state is unknown, there is nothing to be done
	if state, err := c.Status(); err != nil || state == api.StatusA {
		c.expectedEnableUnpluggedState = enable
		return nil
	}

	// if we disable charging with a potential but not yet known communication standard ISO15118
	// this would set allowed A value to be 0. And this would trigger ISO connections to switch to IEC!
	if !enable {
		comStandard, err := c.uc.EvCC.CommunicationStandard(c.evEntity())
		if err != nil || comStandard == evcc.EVCCCommunicationStandardUnknown {
			return api.ErrMustRetry
		}
	}

	var current float64

	if enable {
		current = c.current
	}

	return c.writeCurrentLimitData([]float64{current, current, current})
}

// send current charging power limits to the EV
func (c *EEBus) writeCurrentLimitData(currents []float64) error {
	evEntity := c.evEntity()
	if !c.uc.EvCC.EVConnected(evEntity) {
		return errors.New("no ev connected")
	}

	// check if the EVSE supports overload protection limits
	if !c.uc.OpEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		return api.ErrNotAvailable
	}

	_, maxLimits, _, err := c.uc.OpEV.CurrentLimits(evEntity)
	if err != nil {
		return errors.New("no limits available")
	}

	// setup the limit data structure
	limits := []ucapi.LoadLimitsPhase{}
	for phase, current := range currents {
		if phase >= len(maxLimits) || phase >= len(ucapi.PhaseNameMapping) {
			continue
		}

		limit := ucapi.LoadLimitsPhase{
			Phase:    ucapi.PhaseNameMapping[phase],
			IsActive: true,
			Value:    current,
		}

		// if the limit equals to the max allowed, then the obligation limit is actually inactive
		if current >= maxLimits[phase] {
			limit.IsActive = false
		}

		limits = append(limits, limit)
	}

	// if VAS VW is available, limits are completely covered by it
	// this way evcc can fully control the charging behaviour
	if c.writeLoadControlLimitsVASVW(limits) {
		return nil
	}

	// make sure the recommendations are inactive, otherwise the EV won't go to sleep
	if recommendations, err := c.uc.OscEV.LoadControlLimits(evEntity); err == nil {
		var writeNeeded bool

		for _, item := range recommendations {
			if item.IsActive {
				item.IsActive = false
				writeNeeded = true
			}
		}

		if writeNeeded {
			_, _ = c.uc.OscEV.WriteLoadControlLimits(evEntity, recommendations, nil)
		}
	}

	// Set overload protection limits
	if _, err = c.uc.OpEV.WriteLoadControlLimits(evEntity, limits, nil); err == nil {
		c.currentLimit = currents[0]
	}

	return err
}

// returns if the connected EV has an active VW PV mode
// in this mode, the EV does not have an active charging demand
func (c *EEBus) hasActiveVASVW() bool {
	// EVSE has to support VW VAS
	if !c.vasVW {
		return false
	}

	evEntity := c.evEntity()
	if evEntity == nil {
		return false
	}

	// ISO15118-2 has to be used between EVSE and EV
	if comStandard, err := c.uc.EvCC.CommunicationStandard(evEntity); err != nil || comStandard != model.DeviceConfigurationKeyValueStringTypeISO151182ED2 {
		return false
	}

	// SoC has to be available, otherwise it is plain ISO15118-2
	if _, err := c.Soc(); err != nil {
		return false
	}

	// Optimization of self consumption use case support has to be available
	if !c.uc.EvSoc.IsScenarioAvailableAtEntity(evEntity, 1) {
		return false
	}

	// the use case has to be reported as active
	// only then the EV has no active charging demand and will charge based on OSCEV recommendations
	// this is a workaround for EVSE changing isActive to false, even though they should
	// not announce the usecase at all in that case
	ucs := evEntity.Device().UseCases()
	for _, item := range ucs {
		// check if the referenced entity address is identical to the ev entity address
		// the address may not exist, as it only available since SPINE 1.3
		if item.Address != nil &&
			evEntity.Address() != nil &&
			slices.Compare(item.Address.Entity, evEntity.Address().Entity) != 0 {
			continue
		}

		for _, uc := range item.UseCaseSupport {
			if uc.UseCaseName != nil &&
				*uc.UseCaseName == model.UseCaseNameTypeOptimizationOfSelfConsumptionDuringEVCharging &&
				uc.UseCaseAvailable != nil &&
				*uc.UseCaseAvailable == true {
				return true
			}
		}
	}

	return false
}

// provides support for the special VW VAS ISO15118-2 charging behaviour if supported
// will return false if it isn't supported or successful
//
// this functionality allows to fully control charging without the EV actually having a
// charging demand by itself
func (c *EEBus) writeLoadControlLimitsVASVW(limits []ucapi.LoadLimitsPhase) bool {
	if !c.hasActiveVASVW() {
		return false
	}

	evEntity := c.evEntity()
	if evEntity == nil {
		return false
	}

	// check if the EVSE supports optimization of self consumption limits
	if !c.uc.OscEV.IsScenarioAvailableAtEntity(evEntity, 1) {
		return false
	}

	// on OSCEV all limits have to be active except they are set to the default value
	minLimit, _, _, err := c.uc.OscEV.CurrentLimits(evEntity)
	if err != nil {
		return false
	}

	for index, item := range limits {
		limits[index].IsActive = item.Value >= minLimit[index]
	}

	// send the write command
	if _, err := c.uc.OscEV.WriteLoadControlLimits(evEntity, limits, nil); err != nil {
		return false
	}
	c.currentLimit = limits[0].Value

	// make sure the obligations are inactive, otherwise the EV won't go to sleep
	if obligations, err := c.uc.OpEV.LoadControlLimits(evEntity); err == nil {
		writeNeeded := false

		for index, item := range obligations {
			if item.IsActive {
				obligations[index].IsActive = false
				writeNeeded = true
			}
		}

		if writeNeeded {
			_, _ = c.uc.OpEV.WriteLoadControlLimits(evEntity, obligations, nil)
		}
	}

	return true
}

// MaxCurrent implements the api.Charger interface
func (c *EEBus) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EEBus)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *EEBus) MaxCurrentMillis(current float64) error {
	if !c.connected || c.evEntity() == nil {
		return errors.New("can't set new current as ev is unplugged")
	}

	if err := c.writeCurrentLimitData([]float64{current, current, current}); err != nil {
		return err
	}

	c.current = current

	return nil
}

var _ api.CurrentGetter = (*EEBus)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *EEBus) GetMaxCurrent() (float64, error) {
	if c.currentLimit == -1 {
		return 0, api.ErrNotAvailable
	}

	return c.currentLimit, nil
}

// CurrentPower implements the api.Meter interface
func (c *EEBus) currentPower() (float64, error) {
	evEntity := c.evEntity()
	if evEntity == nil {
		return 0, nil
	}

	var powers []float64

	// does the EVSE provide power data?
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

	var power float64
	for _, phasePower := range powers {
		power += phasePower
	}

	return power, nil
}

// ChargedEnergy implements the api.ChargeRater interface
func (c *EEBus) chargedEnergy() (float64, error) {
	evEntity := c.evEntity()
	if evEntity == nil {
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
	evEntity := c.evEntity()
	if evEntity == nil {
		return 0, 0, 0, nil
	}

	// check if the EVSE supports currents
	if !c.uc.EvCem.IsScenarioAvailableAtEntity(evEntity, 1) {
		return 0, 0, 0, api.ErrNotAvailable
	}

	res, err := c.uc.EvCem.CurrentPerPhase(evEntity)
	if err != nil {
		if err == eebusapi.ErrDataNotAvailable {
			err = api.ErrNotAvailable
		}
		return 0, 0, 0, err
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
	evEntity := c.evEntity()
	if !c.isConnected() || evEntity == nil {
		return "", nil
	}

	if identification, err := c.uc.EvCC.Identifications(evEntity); err == nil && len(identification) > 0 {
		// return the first identification for now
		// later this could be multiple, e.g. MAC Address and PCID
		return identification[0].Value, nil
	}

	if comStandard, _ := c.uc.EvCC.CommunicationStandard(evEntity); comStandard == model.DeviceConfigurationKeyValueStringTypeIEC61851 {
		return "", nil
	}

	if time.Since(c.connectedTime) < maxIdRequestTimespan {
		return "", api.ErrMustRetry
	}

	return "", nil
}

var _ api.Battery = (*EEBus)(nil)

// Soc implements the api.Vehicle interface
func (c *EEBus) Soc() (float64, error) {
	evEntity := c.evEntity()

	if !c.uc.EvSoc.IsScenarioAvailableAtEntity(evEntity, 1) {
		return 0, api.ErrNotAvailable
	}

	soc, err := c.uc.EvSoc.StateOfCharge(evEntity)
	if err != nil {
		return 0, api.ErrNotAvailable
	}

	return soc, nil
}

var _ loadpoint.Controller = (*EEBus)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *EEBus) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
