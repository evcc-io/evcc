package charger

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/enbility/cemd/emobility"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/eebus"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

const (
	maxIdRequestTimespan         = time.Second * 120
	idleFactor                   = 0.6
	voltage              float64 = 230
)

type EEBus struct {
	ski       string
	emobility emobility.EmobilityI

	log *util.Logger
	lp  loadpoint.API

	communicationStandard emobility.EVCommunicationStandardType

	expectedEnableUnpluggedState bool
	current                      float64

	lastIsChargingCheck  time.Time
	lastIsChargingResult bool

	connected     bool
	connectedC    chan bool
	connectedTime time.Time

	mux sync.Mutex
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
	}{
		ChargedEnergy: true,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(cc.Ski, cc.Ip, cc.Meter, cc.ChargedEnergy)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateEEBus -b *EEBus -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)"

// NewEEBus creates EEBus charger
func NewEEBus(ski, ip string, hasMeter, hasChargedEnergy bool) (api.Charger, error) {
	log := util.NewLogger("eebus")

	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		ski:                   ski,
		log:                   log,
		connectedC:            make(chan bool, 1),
		communicationStandard: emobility.EVCommunicationStandardTypeUnknown,
		current:               6,
	}

	c.emobility = eebus.Instance.RegisterEVSE(ski, ip, c.onConnect, c.onDisconnect, nil)

	err := c.waitForConnection()

	if hasMeter {
		var energyG func() (float64, error)
		if hasChargedEnergy {
			energyG = c.chargedEnergy
		}
		return decorateEEBus(c, c.currentPower, c.currents, energyG), err
	}

	return c, err
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

func (c *EEBus) onConnect(ski string) {
	c.log.TRACE.Println("!! onConnect invoked on ski ", ski)

	c.expectedEnableUnpluggedState = false
	c.setDefaultValues()
	c.setConnected(true)
}

func (c *EEBus) onDisconnect(ski string) {
	c.log.TRACE.Println("!! onDisconnect invoked on ski ", ski)

	c.expectedEnableUnpluggedState = false
	c.setConnected(false)
	c.setDefaultValues()
}

func (c *EEBus) setDefaultValues() {
	c.communicationStandard = emobility.EVCommunicationStandardTypeUnknown
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

func (c *EEBus) setLoadpointMinMaxLimits() {
	if c.lp == nil {
		return
	}

	minLimits, maxLimits, _, err := c.emobility.EVCurrentLimits()
	if err != nil || len(minLimits) == 0 || len(maxLimits) == 0 {
		return
	}

	newMin := minLimits[0]
	newMax := maxLimits[0]

	vehicle := c.lp.GetVehicle()

	if c.lp.GetMinCurrent() != newMin && newMin > 0 && (vehicle == nil || vehicle.OnIdentified().MinCurrent == nil) {
		c.lp.SetMinCurrent(newMin)
	}
	if c.lp.GetMaxCurrent() != newMax && newMax > 0 && (vehicle == nil || vehicle.OnIdentified().MaxCurrent == nil) {
		c.lp.SetMaxCurrent(newMax)
	}
}

// we assume that if any phase current value is > idleFactor * min Current, then charging is active and enabled is true
func (c *EEBus) isCharging() bool { // d *communication.EVSEClientDataType
	// check if an external physical meter is assigned
	// we only want this for configured meters and not for internal meters!
	// right now it works as expected
	if c.lp != nil && c.lp.HasChargeMeter() {
		// we only check ever 10 seconds, maybe we can use the config interval duration
		timeDiff := time.Since(c.lastIsChargingCheck)
		if timeDiff.Seconds() >= 10 {
			c.lastIsChargingCheck = time.Now()
			c.lastIsChargingResult = false
			if c.lp.GetChargePower() > c.lp.GetMinPower()*idleFactor {
				c.lastIsChargingResult = true
				return true
			}
		} else if c.lastIsChargingResult {
			return true
		}
	}

	// The above doesn't (yet) work for built in meters, so check the EEBUS measurements also
	currents, err := c.emobility.EVCurrentsPerPhase()
	if err != nil {
		return false
	}
	limitsMin, _, _, err := c.emobility.EVCurrentLimits()
	if err != nil {
		return false
	}

	for index, phaseCurrent := range currents {
		if len(limitsMin) <= index {
			break
		}
		limitMin := limitsMin[index]
		if phaseCurrent > limitMin*idleFactor {
			return true
		}
	}

	return false
}

func (c *EEBus) updateState() (api.ChargeStatus, error) {
	if !c.isConnected() {
		return api.StatusNone, api.ErrTimeout
	}

	if !c.emobility.EVConnected() {
		c.expectedEnableUnpluggedState = false
		return api.StatusA, nil
	}

	currentState, err := c.emobility.EVCurrentChargeState()
	if err != nil {
		return api.StatusNone, err
	}

	switch currentState {
	case emobility.EVChargeStateTypeUnknown, emobility.EVChargeStateTypeUnplugged: // Unplugged
		c.expectedEnableUnpluggedState = false
		return api.StatusA, nil
	case emobility.EVChargeStateTypeFinished, emobility.EVChargeStateTypePaused: // Finished, Paused
		return api.StatusB, nil
	case emobility.EVChargeStateTypeActive: // Active
		if c.isCharging() {
			return api.StatusC, nil
		}
		return api.StatusB, nil
	case emobility.EVChargeStateTypeError: // Error
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("%s properties unknown result: %s", c.ski, currentState)
	}
}

// Status implements the api.Charger interface
func (c *EEBus) Status() (api.ChargeStatus, error) {
	// check the current limits and update if necessary
	c.setLoadpointMinMaxLimits()

	return c.updateState()
}

// Enabled implements the api.Charger interface
// should return true if the charger allows the EV to draw power
func (c *EEBus) Enabled() (bool, error) {
	// when unplugged there is no overload limit data available
	state, err := c.updateState()
	if err != nil || state == api.StatusA {
		return c.expectedEnableUnpluggedState, nil
	}

	limits, err := c.emobility.EVLoadControlObligationLimits()
	if err != nil {
		// there are no overload protection limits available, e.g. because the data was not received yet
		return true, nil
	}

	for _, limit := range limits {
		// for IEC61851 the pause limit is 0A, for ISO15118-2 it is 0.1A
		// instead of checking for the actual data, hardcode this, so we might run into less
		// timing issues as the data might not be received yet
		if limit >= 1 {
			return true, nil
		}
	}

	return false, nil
}

// Enable implements the api.Charger interface
func (c *EEBus) Enable(enable bool) error {
	// if the ev is unplugged or the state is unknown, there is nothing to be done
	if state, err := c.updateState(); err != nil || state == api.StatusA {
		c.expectedEnableUnpluggedState = enable
		return nil
	}

	// if we disable charging with a potential but not yet known communication standard ISO15118
	// this would set allowed A value to be 0. And this would trigger ISO connections to switch to IEC!
	if !enable {
		comStandard, err := c.emobility.EVCommunicationStandard()
		if err != nil || comStandard == emobility.EVCommunicationStandardTypeUnknown {
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
	comStandard, err := c.emobility.EVCommunicationStandard()
	if err != nil {
		return err
	}

	// Only send currents smaller 6A if the communication standard is known
	// otherwise this could cause ISO15118 capable OBCs to stick with IEC61851 when plugging
	// the charge cable in. Or even worse show an error and the cable needs the unplugged,
	// wait for the car to go into sleep and plug it back in.
	// So if are currentls smaller 6A with unknown communication standard change them to 6A
	// keep in mind, that still will confuse evcc as it thinks charging is stopped, but it isn't yet
	if comStandard == emobility.EVCommunicationStandardTypeUnknown {
		minLimits, _, _, err := c.emobility.EVCurrentLimits()
		if err == nil {
			for index, current := range currents {
				if index < len(minLimits) && current < minLimits[index] {
					currents[index] = minLimits[index]
				}
			}
		}
	}

	// set overload protection limits and self consumption limits to identical values
	// so if the EV supports self consumption it will be used automatically
	return c.emobility.EVWriteLoadControlLimits(currents, currents)
}

// MaxCurrent implements the api.Charger interface
func (c *EEBus) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EEBus)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *EEBus) MaxCurrentMillis(current float64) error {
	if !c.connected || !c.emobility.EVConnected() {
		return errors.New("can't set new current as ev is unplugged")
	}

	if err := c.writeCurrentLimitData([]float64{current, current, current}); err != nil {
		return err
	}

	c.current = current

	return nil
}

// CurrentPower implements the api.Meter interface
func (c *EEBus) currentPower() (float64, error) {
	if !c.emobility.EVConnected() {
		return 0, nil
	}

	connectedPhases, err := c.emobility.EVConnectedPhases()
	if err != nil {
		return 0, err
	}

	powers, err := c.emobility.EVPowerPerPhase()
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
	if !c.emobility.EVConnected() {
		return 0, nil
	}

	energy, err := c.emobility.EVChargedEnergy()
	if err != nil {
		return 0, err
	}

	// return kWh
	energy /= 1000

	return energy, nil
}

// Currents implements the api.PhaseCurrents interface
func (c *EEBus) currents() (float64, float64, float64, error) {
	if !c.emobility.EVConnected() {
		return 0, 0, 0, nil
	}

	currents, err := c.emobility.EVCurrentsPerPhase()
	if err != nil {
		return 0, 0, 0, err
	}

	count := len(currents)
	if count < 3 {
		for fill := 0; fill < 3-count; fill++ {
			currents = append(currents, 0)
		}
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Identifier = (*EEBus)(nil)

// Identify implements the api.Identifier interface
func (c *EEBus) Identify() (string, error) {
	if !c.isConnected() || !c.emobility.EVConnected() {
		return "", nil
	}

	if !c.emobility.EVConnected() {
		return "", nil
	}
	if identification, _ := c.emobility.EVIdentification(); identification != "" {
		return identification, nil
	}

	if comStandard, _ := c.emobility.EVCommunicationStandard(); comStandard == emobility.EVCommunicationStandardTypeIEC61851 {
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
	if socSupported, err := c.emobility.EVSoCSupported(); err != nil || !socSupported {
		return 0, api.ErrNotAvailable
	}

	soc, err := c.emobility.EVSoC()
	if err != nil {
		return 0, api.ErrNotAvailable
	}

	return soc, nil
}

var _ loadpoint.Controller = (*EEBus)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *EEBus) LoadpointControl(lp loadpoint.API) {
	c.lp = lp

	c.setLoadpointMinMaxLimits()
}
