package meter

import (
	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/eebus-go/usecases/eg/lpp"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/server/eebus"
)

var _ eebus.Device = (*EEBus)(nil)

// Connect implements the eebus.Device interface.
// On SHIP/SPINE disconnect we drop cached remote-entity references so a
// subsequent re-pair re-populates them from fresh UseCaseSupportUpdate events.
// Without this, Power/Currents/Voltages would keep serving the last value of
// an orphaned entity (see https://github.com/evcc-io/evcc/issues/28518).
func (c *EEBus) Connect(connected bool) {
	c.connector.Connect(connected)

	if connected {
		return
	}

	c.maEntity.Set(nil)
	c.egLpcEntity.Set(nil)
	c.egLppEntity.Set(nil)
}

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	// device removal fires support-update events with a nil entity
	if entity == nil {
		return
	}

	switch event {
	// Monitoring Appliance
	case mpc.UseCaseSupportUpdate, mgcp.UseCaseSupportUpdate:
		c.maEntity.Update(entity)

	// Energy Guard - LPC
	case lpc.UseCaseSupportUpdate:
		c.egLpcEntity.Update(entity)

	// Energy Guard - LPP
	case lpp.UseCaseSupportUpdate:
		c.egLppEntity.Update(entity)
	}
}
