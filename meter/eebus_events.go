package meter

import (
	"slices"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/eebus-go/usecases/eg/lpp"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/server/eebus"
)

var _ eebus.Device = (*EEBus)(nil)

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	switch event {
	// Monitoring Appliance
	case mpc.UseCaseSupportUpdate, mgcp.UseCaseSupportUpdate:
		c.maUseCaseSupportUpdate(entity)
	case mpc.DataUpdatePower, mgcp.DataUpdatePower:
		c.maAssertEntity(entity, c.maDataUpdatePower)
	case mpc.DataUpdateEnergyConsumed, mgcp.DataUpdateEnergyConsumed:
		c.maAssertEntity(entity, c.maDataUpdateEnergyConsumed)
	case mpc.DataUpdateCurrentsPerPhase, mgcp.DataUpdateCurrentPerPhase:
		c.maAssertEntity(entity, c.maDataUpdateCurrentPerPhase)
	case mpc.DataUpdateVoltagePerPhase, mgcp.DataUpdateVoltagePerPhase:
		c.maAssertEntity(entity, c.maDataUpdateVoltagePerPhase)

	// Energy Guard - LPC
	case lpc.UseCaseSupportUpdate:
		c.egLpcUseCaseSupportUpdate(entity)
	case lpc.DataUpdateLimit:
		c.egLpcDataUpdateLimit(entity)

	// Energy Guard - LPP
	case lpp.UseCaseSupportUpdate:
		c.egLppUseCaseSupportUpdate(entity)
	case lpp.DataUpdateLimit:
		c.egLppDataUpdateLimit(entity)
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

// maAssertEntity ignores foreign updates
func (c *EEBus) maAssertEntity(entity spineapi.EntityRemoteInterface, update func(entity spineapi.EntityRemoteInterface)) {
	if c.maEntity == nil || !slices.Equal(entity.Address().Entity, c.maEntity.Address().Entity) {
		return
	}
	update(entity)
}

func (c *EEBus) maDataUpdatePower(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.Power(entity)
	if err != nil {
		c.log.ERROR.Println("Power:", err)
		return
	}
	c.log.TRACE.Printf("Power: %.0fW", data)
	c.power.Set(data)
}

func (c *EEBus) maDataUpdateEnergyConsumed(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.EnergyConsumed(entity)
	if err != nil {
		c.log.ERROR.Println("EnergyConsumed:", err)
		return
	}
	c.log.TRACE.Printf("EnergyConsumed: %.1fkWh", data/1000)
	// Convert Wh to kWh
	c.energy.Set(data / 1000)
}

func (c *EEBus) maDataUpdateCurrentPerPhase(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.CurrentPerPhase(entity)
	if err != nil {
		c.log.ERROR.Println("CurrentPerPhase:", err)
		return
	}
	c.currents.Set(data)
}

func (c *EEBus) maDataUpdateVoltagePerPhase(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.VoltagePerPhase(entity)
	if err != nil {
		c.log.ERROR.Println("VoltagePerPhase:", err)
		return
	}
	c.voltages.Set(data)
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

func (c *EEBus) egLpcDataUpdateLimit(entity spineapi.EntityRemoteInterface) {
	limit, err := c.eg.EgLPCInterface.ConsumptionLimit(entity)
	if err != nil {
		c.log.ERROR.Println("EG LPC ConsumptionLimit:", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.consumptionLimit = limit
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

func (c *EEBus) egLppDataUpdateLimit(entity spineapi.EntityRemoteInterface) {
	limit, err := c.eg.EgLPPInterface.ProductionLimit(entity)
	if err != nil {
		c.log.ERROR.Println("EG LPP ProductionLimit:", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.productionLimit = limit
}
