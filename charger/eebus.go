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

	maxCurrent          float64
	expectedEnableState bool

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
	}

	c.emobility = eebus.Instance.Register(ski, ip, c.onConnect, c.onDisconnect)

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
	timeout := time.After(30 * time.Second)
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

	c.setDefaultValues()
	c.setConnected(true)
}

func (c *EEBus) onDisconnect(ski string) {
	c.log.TRACE.Println("!! onDisconnect invoked on ski ", ski)

	c.expectedEnableState = false
	c.setConnected(false)
	c.setDefaultValues()
}

func (c *EEBus) setDefaultValues() {
	c.expectedEnableState = false
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

	if c.lp.GetMinCurrent() != newMin && newMin > 0 {
		c.lp.SetMinCurrent(newMin)
	}
	if c.lp.GetMaxCurrent() != newMax && newMax > 0 {
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
	currentState, err := c.emobility.EVCurrentChargeState()
	if err != nil {
		return api.StatusNone, err
	}

	if !c.isConnected() {
		return api.StatusNone, fmt.Errorf("%s charger reported as disconnected", c.ski)
	}

	switch currentState {
	case emobility.EVChargeStateTypeUnknown, emobility.EVChargeStateTypeUnplugged: // Unplugged
		c.expectedEnableState = false
		return api.StatusA, nil
	case emobility.EVChargeStateTypeFinished, emobility.EVChargeStateTypePaused: // Finished, Paused
		return api.StatusB, nil
	case emobility.EVChargeStateTypeActive: // Active
		if c.isCharging() {
			// we might already be enabled and charging due to connection issues
			c.expectedEnableState = true
			return api.StatusC, nil
		}
		return api.StatusB, nil
	case emobility.EVChargeStateTypeError: // Error
		return api.StatusF, nil
	}

	return api.StatusNone, fmt.Errorf("%s properties unknown result: %s", c.ski, currentState)
}

// Status implements the api.Charger interface
func (c *EEBus) Status() (api.ChargeStatus, error) {
	return c.updateState()
}

// Enabled implements the api.Charger interface
// should return true if the charger allows the EV to draw power
func (c *EEBus) Enabled() (bool, error) {
	_, err := c.updateState()
	return c.expectedEnableState, err
}

// Enable implements the api.Charger interface
func (c *EEBus) Enable(enable bool) error {
	currentState, err := c.emobility.EVCurrentChargeState()
	if err != nil || currentState == emobility.EVChargeStateTypeUnplugged {
		// if the ev is unplugged, we do not need to disable charging by setting a current of 0 as it already is
		if !enable {
			return nil
		}
		// if the ev is unplugged, we can not enable charging
		return errors.New("can not enable charging as ev is unplugged")
	}

	// if we disable charging with a potential but not yet known communication standard ISO15118
	// this would set allowed A value to be 0. And this would trigger ISO connections to switch to IEC!
	comStandard, err := c.emobility.EVCommunicationStandard()
	if err != nil || comStandard == emobility.EVCommunicationStandardTypeUnknown {
		return api.ErrMustRetry
	}

	// we have to know the limits
	minLimits, maxLimits, _, err := c.emobility.EVCurrentLimits()
	if err != nil {
		return api.ErrMustRetry
	}

	c.expectedEnableState = enable

	if !enable {
		// Important notes on enabling/disabling!!
		// ISO15118 mode:
		//   non-asymmetric or all phases set to 0: the OBC will wait for 1 minute, if the values remain after 1 min, it will pause then
		//   asymmetric and only some phases set to 0: no pauses or waiting for changes required
		//   asymmetric mode requires Plug & Charge (PnC) and Value Added Services (VAS)
		// IEC61851 mode:
		//   switching between 1/3 phases: stop charging, pause for 2 minutes, change phases, resume charging
		//   frequent switching should be avoided by all means!
		c.maxCurrent = 0
		return c.writeCurrentLimitData([]float64{0, 0, 0})
	}

	// if we set MaxCurrent > Min value and then try to enable the charger, it would reset it to min
	if c.maxCurrent > 0 {
		return c.writeCurrentLimitData([]float64{c.maxCurrent, c.maxCurrent, c.maxCurrent})
	}

	// we need to check if the mode is set to now as the currents won't be adjusted afterwards any more in all cases
	if c.lp.GetMode() == api.ModeNow {
		return c.writeCurrentLimitData(maxLimits)
	}

	// in non now mode only enable with min settings, so we don't excessively consume power in case it has to be turned of in the next cycle anyways

	return c.writeCurrentLimitData(minLimits)
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
	chargeState, err := c.emobility.EVCurrentChargeState()
	if err != nil {
		return err
	}
	if chargeState == emobility.EVChargeStateTypeUnplugged {
		return errors.New("can't set new current as ev is unplugged")
	}

	c.maxCurrent = current

	currents := []float64{current, current, current}
	return c.writeCurrentLimitData(currents)
}

// CurrentPower implements the api.Meter interface
func (c *EEBus) currentPower() (float64, error) {
	chargeState, err := c.emobility.EVCurrentChargeState()
	if err != nil {
		return 0, err
	}
	if chargeState == emobility.EVChargeStateTypeUnplugged {
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
	chargeState, err := c.emobility.EVCurrentChargeState()
	if err != nil {
		return 0, err
	}
	if chargeState == emobility.EVChargeStateTypeUnplugged {
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
	chargeState, err := c.emobility.EVCurrentChargeState()
	if err != nil {
		return 0, 0, 0, err
	}
	if chargeState == emobility.EVChargeStateTypeUnplugged {
		return 0, 0, 0, nil
	}

	currents, err := c.emobility.EVCurrentsPerPhase()
	if err != nil {
		return 0, 0, 0, err
	}

	count := len(currents)
	if count < 3 {
		for fill := 0; fill < count-3; fill++ {
			currents = append(currents, 0)
		}
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Identifier = (*EEBus)(nil)

// Identify implements the api.Identifier interface
func (c *EEBus) Identify() (string, error) {
	if !c.isConnected() {
		return "", nil
	}

	chargeState, err := c.emobility.EVCurrentChargeState()
	if err != nil {
		return "", err
	}
	if chargeState == emobility.EVChargeStateTypeUnplugged || chargeState == emobility.EVChargeStateTypeUnknown {
		return "", nil
	}

	identification, err := c.emobility.EVIdentification()
	if err != nil {
		return "", err
	}
	if identification != "" {
		return identification, nil
	}

	comStandard, err := c.emobility.EVCommunicationStandard()
	if err != nil {
		return "", err
	}
	if comStandard == emobility.EVCommunicationStandardTypeIEC61851 {
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
	socSupported, err := c.emobility.EVSoCSupported()
	if err != nil {
		return 0, api.ErrNotAvailable
	}

	if !socSupported {
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
