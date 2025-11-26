package eebus

import (
	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/usecases/cs/lpc"
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
		c.dataUpdateLimit()

	// An incoming load control obligation limit needs to be approved or denied
	//
	// Use `PendingConsumptionLimits` to get the currently pending write approval requests
	// and invoke `ApproveOrDenyConsumptionLimit` for each
	//
	// Use Case LPC, Scenario 1
	case lpc.WriteApprovalRequired:
		c.writeApprovalRequired()

	// Failsafe limit for the consumed active (real) power of the
	// Controllable System data update received
	//
	// Use `FailsafeConsumptionActivePowerLimit` to get the current data
	//
	// Use Case LPC, Scenario 2
	case lpc.DataUpdateFailsafeConsumptionActivePowerLimit:
		c.dataUpdateFailsafeConsumptionActivePowerLimit()

	// Minimum time the Controllable System remains in "failsafe state" unless conditions
	// specified in this Use Case permit leaving the "failsafe state" data update received
	//
	// Use `FailsafeDurationMinimum` to get the current data
	//
	// Use Case LPC, Scenario 2
	case lpc.DataUpdateFailsafeDurationMinimum:
		c.dataUpdateFailsafeDurationMinimum()

	// Indicates a notify heartbeat event the application should care of.
	// E.g. going into or out of the Failsafe state
	//
	// Use Case LPC, Scenario 3
	case lpc.DataUpdateHeartbeat:
		c.dataUpdateHeartbeat()

		// // Load control obligation limit data update received
		// //
		// // Use `ProductionLimit` to get the current data
		// //
		// // Use Case LPC, Scenario 1
		// case lpp.DataUpdateLimit:
		// 	c.dataUpdateLimit()

		// // An incoming load control obligation limit needs to be approved or denied
		// //
		// // Use `PendingProductionLimits` to get the currently pending write approval requests
		// // and invoke `ApproveOrDenyProductionLimit` for each
		// //
		// // Use Case LPC, Scenario 1
		// case lpp.WriteApprovalRequired:
		// 	c.writeApprovalRequired()

		// // Failsafe limit for the produced active (real) power of the
		// // Controllable System data update received
		// //
		// // Use `FailsafeProductionActivePowerLimit` to get the current data
		// //
		// // Use Case LPC, Scenario 2
		// case lpp.DataUpdateFailsafeProductionActivePowerLimit:
		// 	c.dataUpdateFailsafeProductionActivePowerLimit()

		// // Minimum time the Controllable System remains in "failsafe state" unless conditions
		// // specified in this Use Case permit leaving the "failsafe state" data update received
		// //
		// // Use `FailsafeDurationMinimum` to get the current data
		// //
		// // Use Case LPC, Scenario 2
		// case lpp.DataUpdateFailsafeDurationMinimum:
		// 	c.dataUpdateFailsafeDurationMinimum()

		// // Indicates a notify heartbeat event the application should care of.
		// // E.g. going into or out of the Failsafe state
		// //
		// // Use Case LPP, Scenario 3
		// case lpp.DataUpdateHeartbeat:
		// 	c.dataUpdateHeartbeat()
	}
}

func (c *EEBus) dataUpdateLimit() {
	limit, err := c.cs.CsLPCInterface.ConsumptionLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPC ConsumptionLimit:", err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.consumptionLimit = &limit
}

func (c *EEBus) writeApprovalRequired() {
	for msg, limit := range c.cs.CsLPCInterface.PendingConsumptionLimits() {
		c.log.DEBUG.Println("CS LPC PendingConsumptionLimit:", msg, limit)
		if limit.Value < 0 {
			c.cs.CsLPCInterface.ApproveOrDenyConsumptionLimit(msg, false, "negative limit")
			continue
		}

		c.cs.CsLPCInterface.ApproveOrDenyConsumptionLimit(msg, true, "")

		c.mux.Lock()
		c.consumptionLimit = &limit
		c.mux.Unlock()
	}
}

func (c *EEBus) dataUpdateFailsafeConsumptionActivePowerLimit() {
	limit, _, err := c.cs.CsLPCInterface.FailsafeConsumptionActivePowerLimit()
	if err != nil {
		c.log.ERROR.Println("CS LPC FailsafeConsumptionActivePowerLimit:", err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.failsafeLimit = limit
}

func (c *EEBus) dataUpdateFailsafeDurationMinimum() {
	duration, _, err := c.cs.CsLPCInterface.FailsafeDurationMinimum()
	if err != nil {
		c.log.ERROR.Println("CS LPC FailsafeDurationMinimum:", err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	c.failsafeDuration = duration
}

func (c *EEBus) dataUpdateHeartbeat() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.heartbeat.Set(struct{}{})
}

// func (c *EEBus)dataUpdateLimit(){}
// func (c *EEBus)writeApprovalRequired(){}
// func (c *EEBus)dataUpdateFailsafeProductionActivePowerLimit(){}
// func (c *EEBus)dataUpdateFailsafeDurationMinimum(){}
// func (c *EEBus)dataUpdateHeartbeat(){}
