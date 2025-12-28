// Code generated; DO NOT EDIT.

package {{.package}}

import (
	"time"
)

// TODO check state machine against spec
func (c *EEBus) handle{{.Uc}}() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.TRACE.Println("{{.uc}} status:", c.{{.uc}}Status)

	// check heartbeat
	_, heartbeatErr := c.{{.uc}}Heartbeat.Get()
	if heartbeatErr != nil && c.{{.uc}}Status != StatusFailsafe {
		// LPC-914/2
		c.log.WARN.Println("missing {{.uc}} heartbeat- entering failsafe mode")
		c.set{{.Uc}}StatusAndLimit(StatusFailsafe, c.failsafe{{.Uc}}Limit)

		return nil
	}

	// TODO
	// status init
	// status Unlimited/controlled
	// status Unlimited/autonomous

	switch c.{{.uc}}Status {
	case StatusUnlimited:
		// LPC-914/1
		if c.{{.uc}}Limit != nil && c.{{.uc}}Limit.IsActive {
			c.log.WARN.Println("active {{.uc}} limit")
			c.set{{.Uc}}StatusAndLimit(StatusLimited, c.{{.uc}}Limit.Value)
		}

	case StatusLimited:
		// limit updated?
		if !c.{{.uc}}Limit.IsActive {
			c.log.WARN.Println("inactive {{.uc}} limit")
			c.set{{.Uc}}StatusAndLimit(StatusUnlimited, 0)
			break
		}

		// LPC-914/1
		if d := c.{{.uc}}Limit.Duration; d > 0 && time.Since(c.{{.uc}}StatusUpdated) > d {
			c.{{.uc}}Limit.IsActive = false

			c.log.DEBUG.Println("{{.uc}} limit duration exceeded- return to normal")
			c.set{{.Uc}}Limit(0)
		}

	case StatusFailsafe:
		// LPC-914/2
		if d := c.failsafe{{.Uc}}Duration; heartbeatErr == nil || time.Since(c.{{.uc}}StatusUpdated) > d {
			c.log.DEBUG.Println("{{.uc}} heartbeat returned and failsafe duration exceeded- return to normal")
			c.set{{.Uc}}StatusAndLimit(StatusUnlimited, 0)
		}
	}

	return nil
}

func (c *EEBus) set{{.Uc}}StatusAndLimit(status status, limit float64) {
	c.{{.uc}}Status = status
	c.{{.uc}}StatusUpdated = time.Now()

	c.set{{.Uc}}Limit(limit)
}
