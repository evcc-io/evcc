// Code generated; DO NOT EDIT.

package eebus

import (
	"time"
)

// TODO check state machine against spec
func (c *EEBus) handleProduction() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.TRACE.Println("production status:", c.productionStatus)

	// check heartbeat
	_, heartbeatErr := c.productionHeartbeat.Get()
	if heartbeatErr != nil && c.productionStatus != StatusFailsafe {
		// LPC-914/2

		// TODO fix status handling
		c.log.WARN.Println("missing production heartbeat- entering failsafe mode")
		c.setProductionStatusAndLimit(StatusFailsafe, c.failsafeProductionLimit)

		return nil
	}

	// TODO
	// status init
	// status Unlimited/controlled
	// status Unlimited/autonomous

	switch c.productionStatus {
	case StatusUnlimited:
		// LPC-914/1
		if c.productionLimit != nil && c.productionLimit.IsActive {
			c.log.WARN.Println("active production limit")
			c.setProductionStatusAndLimit(StatusLimited, c.productionLimit.Value)
		}

	case StatusLimited:
		// limit updated?
		if !c.productionLimit.IsActive {
			c.log.WARN.Println("inactive production limit")
			c.setProductionStatusAndLimit(StatusUnlimited, 0)
			break
		}

		// LPC-914/1
		if d := c.productionLimit.Duration; d > 0 && time.Since(c.productionStatusUpdated) > d {
			c.productionLimit.IsActive = false

			c.log.DEBUG.Println("production limit duration exceeded- return to normal")
			c.setProductionLimit(0)
		}

	case StatusFailsafe:
		// LPC-914/2
		if d := c.failsafeProductionDuration; heartbeatErr == nil || time.Since(c.productionStatusUpdated) > d {
			c.log.DEBUG.Println("production heartbeat returned and failsafe duration exceeded- return to normal")
			c.setProductionStatusAndLimit(StatusUnlimited, 0)
		}
	}

	return nil
}

func (c *EEBus) setProductionStatusAndLimit(status status, limit float64) {
	c.productionStatus = status
	c.productionStatusUpdated = time.Now()

	c.setProductionLimit(limit)
}
