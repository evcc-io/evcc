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

	c.mu.Lock()
	defer c.mu.Unlock()

	c.maEntity = nil
	c.egLpcEntity = nil
	c.egLppEntity = nil
}

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	switch event {
	// Monitoring Appliance
	case mpc.UseCaseSupportUpdate, mgcp.UseCaseSupportUpdate:
		c.maUseCaseSupportUpdate(entity)

	// Energy Guard - LPC
	case lpc.UseCaseSupportUpdate:
		c.egLpcUseCaseSupportUpdate(entity)

	// Energy Guard - LPP
	case lpp.UseCaseSupportUpdate:
		c.egLppUseCaseSupportUpdate(entity)
	}
}

//
// Monitoring Appliance - MPC/MGPC
//

func (c *EEBus) maUseCaseSupportUpdate(entity spineapi.EntityRemoteInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// use most specific selector
	if c.maEntity == nil || len(entity.Address().Entity) < len(c.maEntity.Address().Entity) {
		c.maEntity = entity
	}
}

//
// Energy Guard - LPC
//

func (c *EEBus) egLpcUseCaseSupportUpdate(entity spineapi.EntityRemoteInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// use most specific selector
	if c.egLpcEntity == nil || len(entity.Address().Entity) < len(c.egLpcEntity.Address().Entity) {
		c.egLpcEntity = entity
	}
}

//
// Energy Guard - LPP
//

func (c *EEBus) egLppUseCaseSupportUpdate(entity spineapi.EntityRemoteInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// use most specific selector
	if c.egLppEntity == nil || len(entity.Address().Entity) < len(c.egLppEntity.Address().Entity) {
		c.egLppEntity = entity
	}
}
