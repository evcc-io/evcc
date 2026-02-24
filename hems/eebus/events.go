package eebus

import (
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/usecases/cs/lpc"
	"github.com/enbility/eebus-go/usecases/cs/lpp"
	spineapi "github.com/enbility/spine-go/api"
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
	case lpc.WriteApprovalRequired:
		c.consumptionWriteApprovalRequired()

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
	case lpp.WriteApprovalRequired:
		c.productionWriteApprovalRequired()

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

func (c *EEBus) updateConsumptionLimit() {
	limit, err := c.cs.CsLPCInterface.ConsumptionLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPC ConsumptionLimit:", err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.consumptionLimit = limit
	c.statusUpdated = time.Now()
}

func (c *EEBus) updateProductionLimit() {
	limit, err := c.cs.CsLPPInterface.ProductionLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPP ProductionLimit:", err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.productionLimit = limit
	c.statusUpdated = time.Now()
}

func (c *EEBus) consumptionWriteApprovalRequired() {
	for msg, limit := range c.cs.CsLPCInterface.PendingConsumptionLimits() {
		c.log.DEBUG.Println("CS LPC PendingConsumptionLimit:", msg, limit)
		if limit.Value < 0 {
			c.cs.CsLPCInterface.ApproveOrDenyConsumptionLimit(msg, false, "negative limit")
			continue
		}

		c.cs.CsLPCInterface.ApproveOrDenyConsumptionLimit(msg, true, "")

		c.mux.Lock()
		c.consumptionLimit = limit
		c.mux.Unlock()
	}
}

func (c *EEBus) productionWriteApprovalRequired() {
	for msg, limit := range c.cs.CsLPPInterface.PendingProductionLimits() {
		c.log.DEBUG.Println("CS LPP PendingProductionLimit:", msg, limit)
		if limit.Value > 0 {
			c.cs.CsLPPInterface.ApproveOrDenyProductionLimit(msg, false, "positive limit")
			continue
		}

		c.cs.CsLPPInterface.ApproveOrDenyProductionLimit(msg, true, "")
		c.mux.Lock()
		c.productionLimit = limit
		c.mux.Unlock()
	}
}

func (c *EEBus) updateFailsafeConsumptionActivePowerLimit() {
	limit, _, err := c.cs.CsLPCInterface.FailsafeConsumptionActivePowerLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPC FailsafeConsumptionActivePowerLimit:", err)
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

	c.mux.Lock()
	defer c.mux.Unlock()

	c.failsafeProductionLimit = limit
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
}
