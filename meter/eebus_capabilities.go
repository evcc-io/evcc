package meter

import (
	"reflect"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/server/eebus"
)

var _ api.Capable = (*EEBus)(nil)

// Capability implements api.Capable. Limit scenarios may arrive after monitoring.
func (c *EEBus) Capability(typ reflect.Type) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch typ {
	case reflect.TypeFor[api.Dimmer]():
		if c.egLpcEntity != nil && c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(c.egLpcEntity, eebus.LPCLimit) {
			return implement.Dimmer(c.dim, c.dimmedState), true
		}
	case reflect.TypeFor[api.Curtailer]():
		if c.egLppEntity != nil && c.eg.EgLPPInterface.IsScenarioAvailableAtEntity(c.egLppEntity, eebus.LPPLimit) {
			return implement.Curtailer(c.curtailedPercent, c.setCurtailPercent), true
		}
	}

	return nil, false
}
