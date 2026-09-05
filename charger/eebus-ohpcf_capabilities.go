package charger

import (
	"reflect"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/server/eebus"
)

// Capability implements api.Capable. LPC may be announced after OHPCF setup.
func (c *EEBusOHPCF) Capability(typ reflect.Type) (any, bool) {
	if typ == reflect.TypeFor[api.Dimmer]() {
		c.mu.RLock()
		defer c.mu.RUnlock()

		if c.egLpcEntity != nil && c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(c.egLpcEntity, eebus.LPCLimit) {
			return implement.Dimmer(c.dim, c.dimmedState), true
		}
		return nil, false
	}

	return c.Caps.Capability(typ)
}
