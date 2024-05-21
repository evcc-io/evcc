package charger

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	cemdapi "github.com/enbility/cemd/api"
	cem "github.com/enbility/cemd/cem"
	"github.com/enbility/cemd/ucevcc"
	cemdutil "github.com/enbility/cemd/util"
	eebusapi "github.com/enbility/eebus-go/api"
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

	expectedEnableUnpluggedState bool
	current                      float64

	// connection tracking for api.CurrentGetter
	evConnected  bool
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
		Meter         bool
		ChargedEnergy bool
	}{
		ChargedEnergy: true,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(cc.Ski, cc.Meter, cc.ChargedEnergy)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateEEBus -b *EEBus -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)"

// NewEEBus creates EEBus charger
func NewEEBus(ski string, hasMeter, hasChargedEnergy bool) (api.Charger, error) {
	log := util.NewLogger("eebus")

	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		ski:        ski,
		log:        log,
		connectedC: make(chan bool, 1),
		current:    6,
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

// Device events
func (c *EEBus) DeviceConnect(device spineapi.DeviceRemoteInterface, event cemdapi.EventType) {
	switch event {
	case cem.DeviceConnected:
		c.onConnect()
	case cem.DeviceDisconnected:
		c.onDisconnect()
	}
}

// UseCase specific events
func (c *EEBus) UseCaseEventCB(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event cemdapi.EventType) {
	switch event {
	// EV
	case ucevcc.EvConnected:
		c.setEvEntity(entity)
	case ucevcc.EvDisconnected:
		c.setEvEntity(nil)
	}
}

func (c *EEBus) onConnect() {
	c.log.TRACE.Println("connect ski:", c.ski)

	c.expectedEnableUnpluggedState = false
	c.setDefaultValues()
	c.setConnected(true)
}

func (c *EEBus) onDisconnect() {
	c.log.TRACE.Println("disconnect ski:", c.ski)

	c.expectedEnableUnpluggedState = false
	c.setConnected(false)
	c.setDefaultValues()
}

func (c *EEBus) setDefaultValues() {
	c.communicationStandard = ucevcc.UCEVCCCommunicationStandardUnknown
	c.lastIsChargingCheck = time.Now().Add(-time.Hour * 1)
	c.lastIsChargingResult = false
}

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
	if !c.uc.EvCC.EVConnected(c.evEntity()) {
		return minMax{}, errors.New("no ev connected")
	}

	minLimits, maxLimits, _, err := c.uc.OpEV.CurrentLimits(c.evEntity())
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
	if !c.uc.EvCC.EVConnected(c.evEntity()) {
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
	currents, err := c.uc.EvCem.CurrentPerPhase(c.evEntity())
	if err != nil {
		return false
	}
	limitsMin, _, _, err := c.uc.OpEV.CurrentLimits(c.evEntity())
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
	if !c.isConnected() {
		return api.StatusNone, api.ErrTimeout
	}

	if !c.uc.EvCC.EVConnected(c.evEntity()) {
		c.expectedEnableUnpluggedState = false
		c.evConnected = false
		return api.StatusA, nil
	}

	if !c.evConnected {
		c.evConnected = true
		c.currentLimit = -1
	}

	currentState, err := c.uc.EvCC.ChargeState(c.ev)
	if err != nil {
		return api.StatusNone, err
	}

	switch currentState {
	case cemdapi.EVChargeStateTypeUnknown, cemdapi.EVChargeStateTypeUnplugged: // Unplugged
		c.expectedEnableUnpluggedState = false
		return api.StatusA, nil
	case cemdapi.EVChargeStateTypeFinished, cemdapi.EVChargeStateTypePaused: // Finished, Paused
		return api.StatusB, nil
	case cemdapi.EVChargeStateTypeActive: // Active
		if c.isCharging() {
			return api.StatusC, nil
		}
		return api.StatusB, nil
	case cemdapi.EVChargeStateTypeError: // Error
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("%s properties unknown result: %s", c.ski, currentState)
	}
}

// Enabled implements the api.Charger interface
// should return true if the charger allows the EV to draw power
func (c *EEBus) Enabled() (bool, error) {
	// when unplugged there is no overload limit data available
	state, err := c.Status()
	if err != nil || state == api.StatusA {
		return c.expectedEnableUnpluggedState, nil
	}

	// if the EV is charging
	if state == api.StatusC {
		return true, nil
	}

	limits, err := c.uc.OpEV.LoadControlLimits(c.evEntity())
	if err != nil {
		// there are no overload protection limits available, e.g. because the data was not received yet
		return true, nil
	}

	for _, limit := range limits {
		// for IEC61851 the pause limit is 0A, for ISO15118-2 it is 0.1A
		// instead of checking for the actual data, hardcode this, so we might run into less
		// timing issues as the data might not be received yet
		if limit.Value >= 1 {
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
		if err != nil || comStandard == ucevcc.UCEVCCCommunicationStandardUnknown {
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
	if !c.uc.EvCC.EVConnected(c.evEntity()) {
		return errors.New("no ev connected")
	}

	comStandard, err := c.uc.EvCC.CommunicationStandard(c.evEntity())
	if err != nil {
		return err
	}

	// Only send currents smaller than 6A if the communication standard is known.
	// Otherwise this could cause ISO15118 capable OBCs to stick with IEC61851 when plugging
	// the charge cable in. Or even worse show an error and the cable then needs to be unplugged,
	// wait for the car to go into sleep and plug it back in.
	// So if there are currents smaller than 6A with unknown communication standard change them to 6A.
	// Keep in mind that this will still confuse evcc as it thinks charging is stopped, but it hasn't yet.
	minLimits, maxLimits, _, err := c.uc.OpEV.CurrentLimits(c.evEntity())
	if err == nil && comStandard == ucevcc.UCEVCCCommunicationStandardUnknown {
		for index, current := range currents {
			if index < len(minLimits) && current < minLimits[index] {
				currents[index] = minLimits[index]
			}
		}
	}

	limits := []cemdapi.LoadLimitsPhase{}
	for phase, current := range currents {
		if phase >= len(maxLimits) || phase >= len(cemdutil.PhaseNameMapping) {
			continue
		}

		limit := cemdapi.LoadLimitsPhase{
			Phase:    cemdutil.PhaseNameMapping[phase],
			IsActive: true,
			Value:    current,
		}

		// if the limit equals to the max allowed, then the limit is actually inactive
		if current >= maxLimits[phase] {
			limit.IsActive = false
		}
	}

	// Set overload protection limits
	if _, err = c.uc.OpEV.WriteLoadControlLimits(c.evEntity(), limits); err == nil {
		c.currentLimit = currents[0]
	}

	// Also set the self consumption limit if available, this makes sure the current behaviour is identical,
	// but in the future this should be changed
	if ok, err := c.uc.OscEV.IsUseCaseSupported(c.evEntity()); err == nil && ok {
		_, _ = c.uc.OscEV.WriteLoadControlLimits(c.evEntity(), limits)
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
	return c.currentLimit, nil
}

// CurrentPower implements the api.Meter interface
func (c *EEBus) currentPower() (float64, error) {
	if c.evEntity() == nil {
		return 0, nil
	}

	connectedPhases, err := c.uc.EvCem.PhasesConnected(c.evEntity())
	if err != nil {
		return 0, err
	}

	powers, err := c.uc.EvCem.PowerPerPhase(c.evEntity())
	if err != nil {
		return 0, err
	}

	var power float64
	for index, phasePower := range powers {
		if index >= int(connectedPhases) {
			break
		}
		power += phasePower
	}

	return power, nil
}

// ChargedEnergy implements the api.ChargeRater interface
func (c *EEBus) chargedEnergy() (float64, error) {
	if c.evEntity() == nil {
		return 0, nil
	}

	energy, err := c.uc.EvCem.EnergyCharged(c.evEntity())
	if err != nil {
		return 0, err
	}

	return energy / 1e3, nil
}

// Currents implements the api.PhaseCurrents interface
func (c *EEBus) currents() (float64, float64, float64, error) {
	if c.evEntity() == nil {
		return 0, 0, 0, nil
	}

	res, err := c.uc.EvCem.CurrentPerPhase(c.evEntity())
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
	if !c.isConnected() || c.evEntity() == nil {
		return "", nil
	}

	if identification, err := c.uc.EvCC.Identifications(c.evEntity()); err == nil && len(identification) > 0 {
		// return the first identification for now
		// later this could be multiple, e.g. MAC Address and PCID
		return identification[0].Value, nil
	}

	if comStandard, _ := c.uc.EvCC.CommunicationStandard(c.evEntity()); comStandard == model.DeviceConfigurationKeyValueStringTypeIEC61851 {
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
	if ok, err := c.uc.EVSoc.IsUseCaseSupported(c.evEntity()); err != nil || !ok {
		return 0, api.ErrNotAvailable
	}

	soc, err := c.uc.EVSoc.StateOfCharge(c.evEntity())
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
