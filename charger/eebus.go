package charger

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/eebus/app"
	"github.com/evcc-io/eebus/communication"
	"github.com/evcc-io/eebus/ship"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
)

const (
	maxIdRequestTimespan = time.Second * 120
	idleFactor           = 0.6
)

type EEBus struct {
	log *util.Logger
	cc  *communication.ConnectionController
	lp  loadpoint.API

	communicationStandard           communication.EVCommunicationStandardEnumType
	socSupportAvailable             bool
	selfConsumptionSupportAvailable bool

	maxCurrent          float64
	connected           bool
	expectedEnableState bool

	lastIsChargingCheck  time.Time
	lastIsChargingResult bool

	evConnectedTime time.Time
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

//go:generate go run ../cmd/tools/decorate.go -f decorateEEBus -b *EEBus -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)"

// NewEEBus creates EEBus charger
func NewEEBus(ski, ip string, hasMeter, hasChargedEnergy bool) (api.Charger, error) {
	log := util.NewLogger("eebus")

	if server.EEBusInstance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:                   log,
		communicationStandard: communication.EVCommunicationStandardEnumTypeUnknown,
	}

	server.EEBusInstance.Register(ski, ip, c.onConnect, c.onDisconnect)

	if hasMeter {
		if hasChargedEnergy {
			return decorateEEBus(c, c.currentPower, c.currents, c.chargedEnergy), nil
		} else {
			return decorateEEBus(c, c.currentPower, c.currents, nil), nil
		}
	}

	return c, nil
}

func (c *EEBus) onConnect(ski string, conn ship.Conn) error {
	c.log.TRACE.Println("!! onConnect invoked on ski ", ski)

	eebusDevice := app.HEMS(server.EEBusInstance.DeviceInfo())
	c.cc = communication.NewConnectionController(c.log.TRACE, conn, eebusDevice)
	c.cc.SetDataUpdateHandler(c.dataUpdateHandler)
	c.cc.Voltage = 230.0 // TODO value should be provided from site

	c.setDefaultValues()
	c.setConnected(true)

	err := c.cc.Boot()

	return err
}

func (c *EEBus) onDisconnect(ski string) {
	c.log.TRACE.Println("!! onDisconnect invoked on ski ", ski)

	c.setConnected(false)
	c.setDefaultValues()
}

func (c *EEBus) setDefaultValues() {
	c.expectedEnableState = false
	c.communicationStandard = communication.EVCommunicationStandardEnumTypeUnknown
	c.socSupportAvailable = false
	c.selfConsumptionSupportAvailable = false
	c.lastIsChargingCheck = time.Now().Add(-time.Hour * 1)
	c.lastIsChargingResult = false
}

func (c *EEBus) setConnected(connected bool) {
	if connected && !c.connected {
		c.evConnectedTime = time.Now()
	}
	c.connected = connected
}

func (c *EEBus) setLoadpointMinMaxLimits(data *communication.EVSEClientDataType) {
	if c.lp == nil {
		return
	}

	if len(data.EVData.Limits) == 0 {
		return
	}

	newMin := data.EVData.Limits[1].Min
	newMax := data.EVData.Limits[1].Max

	if c.lp.GetMinCurrent() != newMin && newMin > 0 {
		c.lp.SetMinCurrent(newMin)
	}
	if c.lp.GetMaxCurrent() != newMax && newMax > 0 {
		c.lp.SetMaxCurrent(newMax)
	}
}

func (c *EEBus) showCurrentChargingSetup() {
	data, err := c.cc.GetData()
	if err != nil {
		return
	}

	prevComStandard := c.communicationStandard
	prevSoCSupport := c.socSupportAvailable
	prevSelfConsumptionSupport := c.selfConsumptionSupportAvailable

	if prevComStandard != data.EVData.CommunicationStandard {
		c.communicationStandard = data.EVData.CommunicationStandard
		c.log.TRACE.Println("ev-charger-communication changed from ", prevComStandard, " to ", data.EVData.CommunicationStandard)
	}

	if prevSoCSupport != data.EVData.UCSoCAvailable {
		c.socSupportAvailable = data.EVData.UCSoCAvailable
		c.log.TRACE.Println("ev-charger-soc support changed from ", prevSoCSupport, " to ", data.EVData.UCSoCAvailable)
	}

	if prevSelfConsumptionSupport != data.EVData.UCSelfConsumptionAvailable {
		c.selfConsumptionSupportAvailable = data.EVData.UCSelfConsumptionAvailable
		c.log.TRACE.Println("ev-charger-self-consumption-support support changed from ", prevSelfConsumptionSupport, " to ", data.EVData.UCSelfConsumptionAvailable)
	}
}

func (c *EEBus) dataUpdateHandler(dataType communication.EVDataElementUpdateType, data *communication.EVSEClientDataType) {
	// we receive data, so it is connected
	c.setConnected(true)

	prevSelfConsumptionSupport := c.selfConsumptionSupportAvailable
	c.showCurrentChargingSetup()

	switch dataType {
	case communication.EVDataElementUpdateUseCaseSelfConsumption:
		// if availability of self consumption use case changes, resend the current charging limit
		// but only if the support value actually changed
		if prevSelfConsumptionSupport != c.selfConsumptionSupportAvailable {
			if err := c.writeCurrentLimitData([]float64{c.maxCurrent, c.maxCurrent, c.maxCurrent}); err != nil {
				c.log.WARN.Println("failed to send current limit data: ", err)
			}
		}
	// case communication.EVDataElementUpdateUseCaseSoC:
	case communication.EVDataElementUpdateEVConnectionState:
		if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
			c.expectedEnableState = false
		}
		c.setLoadpointMinMaxLimits(data)
	case communication.EVDataElementUpdateCommunicationStandard:
		c.communicationStandard = data.EVData.CommunicationStandard
		c.setLoadpointMinMaxLimits(data)
	case communication.EVDataElementUpdateAsymetricChargingType:
		c.setLoadpointMinMaxLimits(data)
	// case communication.EVDataElementUpdateEVSEOperationState:
	// case communication.EVDataElementUpdateEVChargeState:
	// case communication.EVDataElementUpdateChargingStrategy:
	case communication.EVDataElementUpdateChargingPlanRequired:
		if err := c.writeChargingPlan(); err != nil {
			c.log.INFO.Println("failed to send charging plan: ", err)
		}
	case communication.EVDataElementUpdateConnectedPhases:
		c.setLoadpointMinMaxLimits(data)
	case communication.EVDataElementUpdatePowerLimits:
		c.setLoadpointMinMaxLimits(data)
	case communication.EVDataElementUpdateAmperageLimits:
		c.setLoadpointMinMaxLimits(data)
	}
}

// we assume that if any phase current value is > idleFactor * min Current, then charging is active and enabled is true
func (c *EEBus) isCharging(d *communication.EVSEClientDataType) bool {
	// check if an external physical meter is assigned
	// we only want this for configured meters and not for internal meters!
	// right now it works as expected
	if c.lp != nil && c.lp.HasChargeMeter() {
		// we only check ever 10 seconds, maybe we can use the config interval duration
		timeDiff := time.Since(c.lastIsChargingCheck)
		if timeDiff.Seconds() >= 10.0 {
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
	var phase uint
	for phase = 1; phase <= d.EVData.ConnectedPhases; phase++ {
		if phaseCurrent, ok := d.EVData.Measurements.Current.Load(phase); ok {
			if _, ok := phaseCurrent.(float64); ok {
				if phaseCurrent.(float64) > d.EVData.Limits[phase].Min*idleFactor {
					return true
				}
			}
		}
	}

	return false
}

func (c *EEBus) updateState() (api.ChargeStatus, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return api.StatusNone, err
	}

	currentState := data.EVData.ChargeState

	if !c.connected {
		return api.StatusNone, fmt.Errorf("charger reported as disconnected")
	}

	switch currentState {
	case communication.EVChargeStateEnumTypeUnknown, communication.EVChargeStateEnumTypeUnplugged: // Unplugged
		c.expectedEnableState = false
		return api.StatusA, nil
	case communication.EVChargeStateEnumTypeFinished, communication.EVChargeStateEnumTypePaused: // Finished, Paused
		return api.StatusB, nil
	case communication.EVChargeStateEnumTypeActive: // Active
		if c.isCharging(data) {
			// we might already be enabled and charging due to connection issues
			c.expectedEnableState = true
			return api.StatusC, nil
		}
		return api.StatusB, nil
	case communication.EVChargeStateEnumTypeError: // Error
		return api.StatusF, nil
	}

	return api.StatusNone, fmt.Errorf("properties unknown result: %s", currentState)
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
	data, err := c.cc.GetData()
	if err != nil {
		return err
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
		// if the ev is unplugged, we do not need to disable charging by setting a current of 0 as it already is
		if !enable {
			return nil
		}
		// if the ev is unplugged, we can not enable charging
		return errors.New("can not enable charging as ev is unplugged")
	}

	// if we disable charging with a potential but not yet known communication standard ISO15118
	// this would set allowed A value to be 0. And this would trigger ISO connections to switch to IEC!
	if data.EVData.CommunicationStandard == communication.EVCommunicationStandardEnumTypeUnknown {
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
		return c.writeCurrentLimitData([]float64{0.0, 0.0, 0.0})
	}

	// if we set MaxCurrent > Min value and then try to enable the charger, it would reset it to min
	if c.maxCurrent > 0 {
		return c.writeCurrentLimitData([]float64{c.maxCurrent, c.maxCurrent, c.maxCurrent})
	}

	// we need to check if the mode is set to now as the currents won't be adjusted afterwards any more in all cases
	if c.lp.GetMode() == api.ModeNow {
		return c.writeCurrentLimitData([]float64{data.EVData.Limits[1].Max, data.EVData.Limits[2].Max, data.EVData.Limits[3].Max})
	}

	// in non now mode only enable with min settings, so we don't excessively consume power in case it has to be turned of in the next cycle anyways
	return c.writeCurrentLimitData([]float64{data.EVData.Limits[1].Min, data.EVData.Limits[2].Min, data.EVData.Limits[3].Min})
}

// returns true if the connected EV supports charging recommendation
func (c *EEBus) optimizationSelfConsumptionAvailable() bool {
	data, err := c.cc.GetData()
	if err == nil {
		return data.EVData.UCSelfConsumptionAvailable
	}

	return false
}

// respond to a charging plan request from the EV
func (c *EEBus) writeChargingPlan() error {
	data, err := c.cc.GetData()
	if err != nil {
		return err
	}

	var chargingPlan communication.EVChargingPlan

	tariffGrid := 0.30
	tariffFeedIn := 0.10
	maxPower := c.lp.GetMaxPower()

	switch data.EVData.ChargingStrategy {
	case communication.EVChargingStrategyEnumTypeNoDemand, communication.EVChargingStrategyEnumTypeUnknown:
		// The EV has no power demand or we don't know it yet, so we shouldn't get here
		// TODO: why did we get here?

		// lets do 24 1 hour slots with maximum power, power will be adjusted via Overload Protection limits
		for i := 0; i < 24; i++ {
			chargingPlan.Slots = append(chargingPlan.Slots, communication.EVChargingSlot{
				Duration: time.Hour,
				MaxValue: maxPower,
				Pricing:  tariffGrid,
			})
		}
		chargingPlan.Duration = 24 * time.Hour
	case communication.EVChargingStrategyEnumTypeDirectCharging:
		// The EV is in direct charging mode

		// Does it support self consumption?
		if c.optimizationSelfConsumptionAvailable() {
			// this should mean that any mode in evcc is ignored and the EV is in full control
			// TODO: is this the right approach?

			// lets do one 24 hour slot with maximum power, power will be adjusted via Overload Protection limits
			chargingPlan.Slots = append(chargingPlan.Slots, communication.EVChargingSlot{
				Duration: time.Duration(24) * time.Hour,
				MaxValue: maxPower,
				Pricing:  tariffGrid,
			})
			chargingPlan.Duration = 24 * time.Hour
		} else {
			// in this mode we need to enforce the evcc modes

			// we need to create a 24h charging plan
			chargingPlan.Duration = 24 * time.Hour

			currentMode := c.lp.GetMode()
			switch currentMode {
			case api.ModeNow, api.ModeMinPV:
				// lets do one 24 hour slot with maximum power, power will be adjusted via Overload Protection limits
				chargingPlan.Slots = append(chargingPlan.Slots, communication.EVChargingSlot{
					Duration: time.Duration(24) * time.Hour,
					MaxValue: maxPower,
					Pricing:  tariffGrid,
				})
				chargingPlan.Duration = 24 * time.Hour
			case api.ModePV:
				// lets do 24 1 hour slots with maximum power, power will be adjusted via Overload Protection limits
				// but set the nightly hours to 0 W, we assume those to be from 20:00 to 07:00
				now := time.Now()
				for i := 0; i < 24; i++ {
					power := maxPower
					pricing := tariffFeedIn
					if now.Hour()+i >= 20 || now.Hour()+i < 7 {
						power = 0.0
						pricing = tariffGrid
					}
					chargingPlan.Slots = append(chargingPlan.Slots, communication.EVChargingSlot{
						Duration: time.Hour,
						MaxValue: power,
						Pricing:  pricing,
					})
				}
				chargingPlan.Duration = 24 * time.Hour
			case api.ModeOff:
				// lets do 24 1 hour slots with 0 W, so it wakes at once an hour to check back
				for i := 0; i < 24; i++ {
					chargingPlan.Slots = append(chargingPlan.Slots, communication.EVChargingSlot{
						Duration: time.Hour,
						MaxValue: 0,
						Pricing:  tariffGrid,
					})
				}
				chargingPlan.Duration = 24 * time.Hour
			}
		}
	case communication.EVChargingStrategyEnumTypeTimedCharging:
		// The EV is in timed charging mode

		targetDuration := data.EVData.ChargingTargetDuration

		// split the duration into full hours, with the remaining time at the start
		hours := int(targetDuration.Hours())
		remainingDuration := targetDuration - (time.Duration(hours) * time.Hour)

		if remainingDuration > 0 {
			chargingPlan.Slots = append(chargingPlan.Slots, communication.EVChargingSlot{
				Duration: remainingDuration,
				MaxValue: maxPower,
				Pricing:  tariffGrid,
			})
		}

		for i := 0; i < hours; i++ {
			chargingPlan.Slots = append(chargingPlan.Slots, communication.EVChargingSlot{
				Duration: time.Hour,
				MaxValue: maxPower,
				Pricing:  tariffGrid,
			})
		}
		chargingPlan.Duration = targetDuration

	default:
		return fmt.Errorf("charging strategy not implemented: %s", data.EVData.ChargingStrategy)
	}

	return c.cc.WriteChargingPlan(chargingPlan)
}

// send current charging power limits to the EV
func (c *EEBus) writeCurrentLimitData(currents []float64) error {
	data, err := c.cc.GetData()
	if err != nil {
		return err
	}

	// Only send currents smaller 6A if the communication standard is known
	// otherwise this could cause ISO15118 capable OBCs to stick with IEC61851 when plugging
	// the charge cable in. Or even worse show an error and the cable needs the unplugged,
	// wait for the car to go into sleep and plug it back in.
	// So if are currentls smaller 6A with unknown communication standard change them to 6A
	// keep in mind, that still will confuse evcc as it thinks charging is stopped, but it isn't yet
	if data.EVData.CommunicationStandard == communication.EVCommunicationStandardEnumTypeUnknown {
		for index, current := range currents {
			phase := uint(index) + 1
			if limit, ok := data.EVData.Limits[phase]; ok {
				if current < limit.Min {
					currents[index] = limit.Min
				}
			}
		}
	}

	// set overload protection limits and self consumption limits to identical values
	// so if the EV supports self consumption it will be used automatically
	return c.cc.WriteCurrentLimitData(currents, currents, &data.EVData)
}

// MaxCurrent implements the api.Charger interface
func (c *EEBus) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EEBus)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *EEBus) MaxCurrentMillis(current float64) error {
	data, err := c.cc.GetData()
	if err != nil {
		return err
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
		return errors.New("can't set new current as ev is unplugged")
	}

	// if data.EVData.Limits[1].Min == 0 {
	// 	c.log.TRACE.Println("!! we did not yet receive min and max currents to validate the call of MaxCurrent, use it as is")
	// }

	if current < data.EVData.Limits[1].Min {
		current = data.EVData.Limits[1].Min
	}

	if current > data.EVData.Limits[1].Max {
		current = data.EVData.Limits[1].Max
	}

	c.maxCurrent = current

	// TODO error handling

	currents := []float64{current, current, current}
	return c.writeCurrentLimitData(currents)
}

// CurrentPower implements the api.Meter interface
func (c *EEBus) currentPower() (float64, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return 0, err
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
		return 0, nil
	}

	var power float64
	for phase := uint(1); phase <= data.EVData.ConnectedPhases; phase++ {
		if phasePower, ok := data.EVData.Measurements.Power.Load(phase); ok {
			if _, ok := phasePower.(float64); ok {
				power += phasePower.(float64)
			}
		}
	}

	return power, nil
}

// ChargedEnergy implements the api.ChargeRater interface
func (c *EEBus) chargedEnergy() (float64, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return 0, err
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
		return 0, nil
	}

	energy := data.EVData.Measurements.ChargedEnergy / 1000

	return energy, nil
}

// Currents implements the api.MeterCurrent interface
func (c *EEBus) currents() (float64, float64, float64, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return 0, 0, 0, err
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
		return 0, 0, 0, nil
	}

	var currents []float64

	for phase := uint(1); phase <= 3; phase++ {
		current := 0.0
		if value, ok := data.EVData.Measurements.Current.Load(phase); ok {
			if _, ok := value.(float64); ok {
				current = value.(float64)
			}
		}
		currents = append(currents, current)
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Identifier = (*EEBus)(nil)

// Identify implements the api.Identifier interface
func (c *EEBus) Identify() (string, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return "", err
	}

	if !c.connected {
		return "", nil
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged || data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnknown {
		return "", nil
	}

	if len(data.EVData.Identification) > 0 {
		return data.EVData.Identification, nil
	}

	if data.EVData.CommunicationStandard == communication.EVCommunicationStandardEnumTypeIEC61851 {
		return "", nil
	}

	if time.Since(c.evConnectedTime) < maxIdRequestTimespan {
		return "", api.ErrMustRetry
	}

	return "", nil
}

var _ api.Battery = (*EEBus)(nil)

// SoC implements the api.Vehicle interface
func (c *EEBus) SoC() (float64, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return 0, api.ErrMustRetry
	}

	if !data.EVData.UCSoCAvailable || !data.EVData.SoCDataAvailable {
		return 0, api.ErrNotAvailable
	}

	return data.EVData.Measurements.SoC, nil
}

var _ loadpoint.Controller = (*EEBus)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *EEBus) LoadpointControl(lp loadpoint.API) {
	c.lp = lp

	// set current known min, max current limits
	data, err := c.cc.GetData()
	if err != nil {
		return
	}
	c.setLoadpointMinMaxLimits(data)
	c.showCurrentChargingSetup()
}
