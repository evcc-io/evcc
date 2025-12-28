// Code generated; DO NOT EDIT.

package eebus

import (
	"time"
)

// TODO check state machine against spec
func (c *EEBus) handleConsumption() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.TRACE.Println("consumption status:", c.consumptionStatus)

	// check heartbeat
	_, heartbeatErr := c.consumptionHeartbeat.Get()
	if heartbeatErr != nil && c.consumptionStatus != StatusFailsafe {
		// LPC-914/2

		// TODO fix status handling
		c.log.WARN.Println("missing consumption heartbeat- entering failsafe mode")
		c.setConsumptionStatusAndLimit(StatusFailsafe, c.failsafeConsumptionLimit)

		return nil
	}

	// TODO
	// status init
	// status Unlimited/controlled
	// status Unlimited/autonomous

	switch c.consumptionStatus {
	case StatusUnlimited:
		// LPC-914/1
		if c.consumptionLimit != nil && c.consumptionLimit.IsActive {
			c.log.WARN.Println("active consumption limit")
			c.setConsumptionStatusAndLimit(StatusLimited, c.consumptionLimit.Value)
		}

	case StatusLimited:
		// limit updated?
		if !c.consumptionLimit.IsActive {
			c.log.WARN.Println("inactive consumption limit")
			c.setConsumptionStatusAndLimit(StatusUnlimited, 0)
			break
		}

		// LPC-914/1
		if d := c.consumptionLimit.Duration; d > 0 && time.Since(c.consumptionStatusUpdated) > d {
			c.consumptionLimit.IsActive = false

			c.log.DEBUG.Println("consumption limit duration exceeded- return to normal")
			c.setConsumptionLimit(0)
		}

	case StatusFailsafe:
		// LPC-914/2
		if d := c.failsafeConsumptionDuration; heartbeatErr == nil || time.Since(c.consumptionStatusUpdated) > d {
			c.log.DEBUG.Println("consumption heartbeat returned and failsafe duration exceeded- return to normal")
			c.setConsumptionStatusAndLimit(StatusUnlimited, 0)
		}
	}

	return nil
}

func (c *EEBus) setConsumptionStatusAndLimit(status status, limit float64) {
	c.consumptionStatus = status
	c.consumptionStatusUpdated = time.Now()

	c.setConsumptionLimit(limit)
}
