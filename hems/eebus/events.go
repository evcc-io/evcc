package eebus

import (
	"slices"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cs/lpc"
	"github.com/enbility/eebus-go/usecases/cs/lpp"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/server/eebus"
)

var _ eebus.Device = (*EEBus)(nil)

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	switch event {
	// Load control obligation limit data update received
	//
	// Use `ConsumptionLimit` to get the current data
	//
	// Use Case LPC, Scenario 1
	case lpc.DataUpdateLimit:
		c.updateConsumptionLimit()

	// An incoming load control obligation limit needs to be approved or denied
	//
	// Use `PendingConsumptionLimits` to get the currently pending write approval requests
	// and invoke `ApproveOrDenyConsumptionLimit` for each
	//
	// Use Case LPC, Scenario 1
	case lpc.LimitWriteApprovalRequired:
		c.consumptionWriteApprovalRequired()

	// An incoming device configuration write (e.g. failsafe values) needs to be
	// approved or denied. eebus-go <= v0.7.0 applied these automatically, so we
	// keep that behaviour by approving all pending configuration writes.
	//
	// Use Case LPC, Scenario 2
	case lpc.ConfigurationWriteApprovalRequired:
		c.approveDeviceConfigurations(c.cs.CsLPCInterface.PendingDeviceConfigurations(), c.cs.CsLPCInterface.ApproveOrDenyDeviceConfiguration, c.cs.CsLPCInterface.SetFailsafeDurationMinimum)

	// Failsafe limit for the consumed active (real) power of the
	// Controllable System data update received
	//
	// Use `FailsafeConsumptionActivePowerLimit` to get the current data
	//
	// Use Case LPC, Scenario 2
	case lpc.DataUpdateFailsafeConsumptionActivePowerLimit:
		c.updateFailsafeConsumptionActivePowerLimit()

	// Minimum time the Controllable System remains in "failsafe state" unless conditions
	// specified in this Use Case permit leaving the "failsafe state" data update received
	//
	// Use `FailsafeDurationMinimum` to get the current data
	//
	// Use Case LPC, Scenario 2
	case lpc.DataUpdateFailsafeDurationMinimum:
		c.updateFailsafeConsumptionDurationMinimum()

	// Indicates a notify heartbeat event the application should care of.
	// E.g. going into or out of the Failsafe state
	//
	// Use Case LPC, Scenario 3
	case lpc.DataUpdateHeartbeat:
		c.updateHeartbeat()

	// Load control obligation limit data update received
	//
	// Use `ProductionLimit` to get the current data
	//
	// Use Case LPC, Scenario 1
	case lpp.DataUpdateLimit:
		c.updateProductionLimit()

	// An incoming load control obligation limit needs to be approved or denied
	//
	// Use `PendingProductionLimits` to get the currently pending write approval requests
	// and invoke `ApproveOrDenyProductionLimit` for each
	//
	// Use Case LPP, Scenario 1
	case lpp.LimitWriteApprovalRequired:
		c.productionWriteApprovalRequired()

	// An incoming device configuration write (e.g. failsafe values) needs to be
	// approved or denied. eebus-go <= v0.7.0 applied these automatically, so we
	// keep that behaviour by approving all pending configuration writes.
	//
	// Use Case LPP, Scenario 2
	case lpp.ConfigurationWriteApprovalRequired:
		c.approveDeviceConfigurations(c.cs.CsLPPInterface.PendingDeviceConfigurations(), c.cs.CsLPPInterface.ApproveOrDenyDeviceConfiguration, c.cs.CsLPPInterface.SetFailsafeDurationMinimum)

	// Failsafe limit for the produced active (real) power of the
	// Controllable System data update received
	//
	// Use `FailsafeProductionActivePowerLimit` to get the current data
	//
	// Use Case LPP, Scenario 2
	case lpp.DataUpdateFailsafeProductionActivePowerLimit:
		c.updateFailsafeProductionActivePowerLimit()

	// Minimum time the Controllable System remains in "failsafe state" unless conditions
	// specified in this Use Case permit leaving the "failsafe state" data update received
	//
	// Use `FailsafeDurationMinimum` to get the current data
	//
	// Use Case LPP, Scenario 2
	case lpp.DataUpdateFailsafeDurationMinimum:
		c.updateFailsafeProductionDurationMinimum()

	// Indicates a notify heartbeat event the application should care of.
	// E.g. going into or out of the Failsafe state
	//
	// Use Case LPP, Scenario 3
	case lpp.DataUpdateHeartbeat:
		c.updateHeartbeat()
	}
}

// setConsumptionLimitData stores a freshly received LPC limit. eebus-go reports a
// stated duration as the time *remaining* (spine stores an absolute end time), so an
// already-active limit must restart the clock run() measures the duration against.
// Caller holds the mutex.
func (c *EEBus) setConsumptionLimitData(limit ucapi.LoadLimit) {
	c.consumptionLimit = limit
	c.limitReceived = time.Now()

	if limitActive(c.consumptionLimitActivated) {
		*c.consumptionLimitActivated = c.limitReceived
	}
}

// setProductionLimitData stores a freshly received LPP limit, see setConsumptionLimitData.
// Caller holds the mutex.
func (c *EEBus) setProductionLimitData(limit ucapi.LoadLimit) {
	c.productionLimit = limit
	c.limitReceived = time.Now()

	if limitActive(c.productionLimitActivated) {
		*c.productionLimitActivated = c.limitReceived
	}
}

func (c *EEBus) updateConsumptionLimit() {
	limit, err := c.cs.CsLPCInterface.ConsumptionLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPC ConsumptionLimit:", err)
		return
	}

	if invalidLimit(limit.Value) {
		c.log.ERROR.Printf("CS LPC ConsumptionLimit: invalid limit: %.0fW", limit.Value)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.setConsumptionLimitData(limit)
}

func (c *EEBus) updateProductionLimit() {
	limit, err := c.cs.CsLPPInterface.ProductionLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPP ProductionLimit:", err)
		return
	}

	// LPP states its limits as negative watts
	if invalidLimit(-limit.Value) {
		c.log.ERROR.Printf("CS LPP ProductionLimit: invalid limit: %.0fW", limit.Value)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.setProductionLimitData(limit)
}

func (c *EEBus) consumptionWriteApprovalRequired() {
	for msg, limit := range c.cs.CsLPCInterface.PendingConsumptionLimits() {
		c.log.DEBUG.Println("CS LPC PendingConsumptionLimit:", msg, limit)
		if invalidLimit(limit.Value) {
			c.cs.CsLPCInterface.ApproveOrDenyConsumptionLimit(msg, false, "invalid limit")
			continue
		}

		c.mux.Lock()
		accept := c.acceptsLimitWrite()
		if accept {
			c.controlled = true
			c.setConsumptionLimitData(limit)
		}
		c.mux.Unlock()

		if !accept {
			c.log.DEBUG.Println("CS LPC: denying limit without preceding heartbeat:", msg)
			c.cs.CsLPCInterface.ApproveOrDenyConsumptionLimit(msg, false, "no preceding heartbeat")
			continue
		}

		c.cs.CsLPCInterface.ApproveOrDenyConsumptionLimit(msg, true, "")
	}
}

func (c *EEBus) productionWriteApprovalRequired() {
	for msg, limit := range c.cs.CsLPPInterface.PendingProductionLimits() {
		c.log.DEBUG.Println("CS LPP PendingProductionLimit:", msg, limit)
		if invalidLimit(-limit.Value) {
			c.cs.CsLPPInterface.ApproveOrDenyProductionLimit(msg, false, "invalid limit")
			continue
		}

		c.mux.Lock()
		accept := c.acceptsLimitWrite()
		if accept {
			c.controlled = true
			c.setProductionLimitData(limit)
		}
		c.mux.Unlock()

		if !accept {
			c.log.DEBUG.Println("CS LPP: denying limit without preceding heartbeat:", msg)
			c.cs.CsLPPInterface.ApproveOrDenyProductionLimit(msg, false, "no preceding heartbeat")
			continue
		}

		c.cs.CsLPPInterface.ApproveOrDenyProductionLimit(msg, true, "")
	}
}

// failsafe duration minimum range the Energy Guard may choose from ([LPC-022/3], [LPP-022/3])
const (
	failsafeDurationMin = 2 * time.Hour
	failsafeDurationMax = 24 * time.Hour // our maximum, see [LPC-022/4]
)

// approveDeviceConfigurations approves pending device configuration writes. It denies
//   - all writes while the CS is not controlled by the Energy Guard: in "init", "failsafe
//     state" and "unlimited/autonomous" other data points are evaluated only after a
//     heartbeat and a following limit (LPC/LPP UC TS 1.0.0 section 2.2, [LPC-003/2])
//   - writes stating an invalid failsafe limit
//   - a failsafe duration minimum below 2h ([LPC-003/2], [LPC-022/3])
//   - a failsafe duration minimum above our maximum, which then becomes the failsafe
//     duration minimum ([LPC-022/4], [LPC-022/5])
func (c *EEBus) approveDeviceConfigurations(
	pending map[model.MsgCounterType][]ucapi.PendingDeviceConfiguration,
	approve func(msgCounter model.MsgCounterType, approve bool, reason string),
	setDuration func(duration time.Duration, changeable bool) error,
) {
	c.mux.RLock()
	controlled := c.controlled
	c.mux.RUnlock()

	for msg, configs := range pending {
		switch {
		case !controlled:
			c.log.DEBUG.Println("denying device configuration write while not controlled by the energy guard:", msg, configs)
			approve(msg, false, "not controlled by energy guard")

		case slices.ContainsFunc(configs, invalidFailsafeLimit):
			c.log.ERROR.Println("denying device configuration write:", msg, configs)
			approve(msg, false, "invalid limit")

		case slices.ContainsFunc(configs, failsafeDurationTooShort):
			c.log.ERROR.Println("denying failsafe duration below minimum:", msg, configs)
			approve(msg, false, "failsafe duration below minimum")

		case slices.ContainsFunc(configs, failsafeDurationTooLong):
			c.log.ERROR.Println("denying failsafe duration above maximum:", msg, configs)
			approve(msg, false, "failsafe duration above maximum")

			if err := setDuration(failsafeDurationMax, true); err != nil {
				c.log.ERROR.Println("SetFailsafeDurationMinimum:", err)
				continue
			}

			c.mux.Lock()
			c.failsafeDuration = failsafeDurationMax
			c.mux.Unlock()

		default:
			c.log.DEBUG.Println("approving device configuration write:", msg, configs)
			approve(msg, true, "")
		}
	}
}

// failsafeDurationOf returns the failsafe duration minimum a configuration write states
func failsafeDurationOf(config ucapi.PendingDeviceConfiguration) (time.Duration, bool) {
	if config.Value == nil || config.Value.Duration == nil {
		return 0, false
	}
	d, err := config.Value.Duration.GetTimeDuration()
	return d, err == nil
}

// failsafeDurationTooShort reports whether a configuration write states an unreadable
// failsafe duration minimum or one below 2h
func failsafeDurationTooShort(config ucapi.PendingDeviceConfiguration) bool {
	if config.KeyName != model.DeviceConfigurationKeyNameTypeFailsafeDurationMinimum {
		return false
	}
	d, ok := failsafeDurationOf(config)
	return !ok || d < failsafeDurationMin
}

// failsafeDurationTooLong reports whether a configuration write states a failsafe
// duration minimum above our maximum
func failsafeDurationTooLong(config ucapi.PendingDeviceConfiguration) bool {
	if config.KeyName != model.DeviceConfigurationKeyNameTypeFailsafeDurationMinimum {
		return false
	}
	d, ok := failsafeDurationOf(config)
	return ok && d > failsafeDurationMax
}

// invalidFailsafeLimit reports whether a configuration write states an invalid failsafe limit
func invalidFailsafeLimit(config ucapi.PendingDeviceConfiguration) bool {
	switch config.KeyName {
	case model.DeviceConfigurationKeyNameTypeFailsafeConsumptionActivePowerLimit,
		model.DeviceConfigurationKeyNameTypeFailsafeProductionActivePowerLimit:
		return config.Value != nil && config.Value.ScaledNumber != nil && invalidLimit(config.Value.ScaledNumber.GetValue())
	}
	return false
}

func (c *EEBus) updateFailsafeConsumptionActivePowerLimit() {
	limit, _, err := c.cs.CsLPCInterface.FailsafeConsumptionActivePowerLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPC FailsafeConsumptionActivePowerLimit:", err)
		return
	}

	if invalidLimit(limit) {
		c.log.ERROR.Printf("CS LPC FailsafeConsumptionActivePowerLimit: invalid limit: %.0fW", limit)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.failsafeConsumptionLimit = limit
}

func (c *EEBus) updateFailsafeProductionActivePowerLimit() {
	limit, _, err := c.cs.CsLPPInterface.FailsafeProductionActivePowerLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPP FailsafeProductionActivePowerLimit:", err)
		return
	}

	if invalidLimit(limit) {
		c.log.ERROR.Printf("CS LPP FailsafeProductionActivePowerLimit: invalid limit: %.0fW", limit)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.failsafeProductionLimit = new(limit)
}

func (c *EEBus) updateFailsafeConsumptionDurationMinimum() {
	duration, _, err := c.cs.CsLPCInterface.FailsafeDurationMinimum()
	if err != nil {
		c.log.ERROR.Println("CS LPC FailsafeDurationMinimum:", err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.failsafeDuration = duration
}

func (c *EEBus) updateFailsafeProductionDurationMinimum() {
	duration, _, err := c.cs.CsLPPInterface.FailsafeDurationMinimum()
	if err != nil {
		c.log.ERROR.Println("CS LPP FailsafeDurationMinimum:", err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.failsafeDuration = duration
}

func (c *EEBus) updateHeartbeat() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.heartbeat.Set(struct{}{})
	c.heartbeatAt = time.Now()
}
