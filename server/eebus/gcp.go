package eebus

import (
	"fmt"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	gcpmgcp "github.com/enbility/eebus-go/usecases/gcp/mgcp"
	"github.com/enbility/spine-go/model"
	"github.com/enbility/spine-go/spine"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
)

// gcpEntity is the local entity address of the grid connection point. It is separate from the
// CS entity since both register DeviceConfiguration and ElectricalConnection server features.
const gcpEntity = 4

// GridConnectionPoint publishes the grid meter via MGCP with evcc as Grid Connection Point
type GridConnectionPoint struct {
	log                *util.Logger
	uc                 ucapi.GcpMGCPInterface
	currents, voltages bool // scenarios registered at creation; their update helpers panic otherwise
}

// gcpScenarios returns the optional MGCP scenarios the meter supports. The use case requires
// power and both energy directions, ok is false if the meter cannot provide them.
func gcpScenarios(meter api.Meter) (currents, voltages, ok bool) {
	_, energy := api.Cap[api.MeterEnergy](meter)
	_, returnEnergy := api.Cap[api.MeterReturnEnergy](meter)
	if !energy || !returnEnergy {
		return false, false, false
	}

	_, currents = api.Cap[api.PhaseCurrents](meter)
	_, voltages = api.Cap[api.PhaseVoltages](meter)

	return currents, voltages, true
}

// AddGridConnectionPoint announces the MGCP use case for the grid meter. It returns nil
// without announcing anything if the meter lacks the values MGCP requires.
func (c *EEBus) AddGridConnectionPoint(meter api.Meter) (*GridConnectionPoint, error) {
	currents, voltages, ok := gcpScenarios(meter)
	if !ok {
		return nil, nil
	}

	measured := new(model.MeasurementValueSourceTypeMeasuredValue)

	var currentConfig *gcpmgcp.MonitorCurrentConfig
	if currents {
		currentConfig = &gcpmgcp.MonitorCurrentConfig{
			ValueSourcePhaseA: measured,
			ValueSourcePhaseB: measured,
			ValueSourcePhaseC: measured,
		}
	}

	var voltageConfig *gcpmgcp.MonitorVoltageConfig
	if voltages {
		voltageConfig = &gcpmgcp.MonitorVoltageConfig{
			ValueSourcePhaseA: measured,
			ValueSourcePhaseB: measured,
			ValueSourcePhaseC: measured,
		}
	}

	localDevice := c.service.LocalDevice()
	entity := spine.NewEntityLocal(localDevice, model.EntityTypeTypeGridConnectionPointOfPremises, []model.AddressEntityType{gcpEntity}, 4*time.Second)

	uc, err := gcpmgcp.NewMGCP(entity, nil,
		nil,
		&gcpmgcp.MonitorPowerConfig{ValueSource: measured},
		&gcpmgcp.MonitorEnergyConfig{ValueSourceProduction: measured, ValueSourceConsumption: measured},
		currentConfig,
		voltageConfig,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("mgcp: %w", err)
	}

	// the service is already running: announce the entity with its features before the use case,
	// otherwise remotes cannot resolve the use case address and match it to other entities
	if err := uc.AddFeatures(); err != nil {
		return nil, fmt.Errorf("mgcp: %w", err)
	}
	localDevice.AddEntity(entity)
	uc.AddUseCase()

	return &GridConnectionPoint{log: c.log, uc: uc, currents: currents, voltages: voltages}, nil
}

// Update publishes the grid measurement
func (g *GridConnectionPoint) Update(m types.Measurement) {
	now := time.Now()

	data := []ucapi.UpdateData{g.uc.UpdateDataPowerTotal(m.Power, &now, nil)}

	// MGCP energy is in Wh
	if m.Energy != nil {
		data = append(data, g.uc.UpdateDataEnergyConsumed(*m.Energy*1e3, &now, nil, nil, nil))
	}
	if m.ReturnEnergy != nil {
		data = append(data, g.uc.UpdateDataEnergyFeedIn(*m.ReturnEnergy*1e3, &now, nil, nil, nil))
	}

	if g.currents && len(m.Currents) == 3 {
		data = append(data,
			g.uc.UpdateDataCurrentPhaseA(m.Currents[0], &now, nil),
			g.uc.UpdateDataCurrentPhaseB(m.Currents[1], &now, nil),
			g.uc.UpdateDataCurrentPhaseC(m.Currents[2], &now, nil),
		)
	}

	if g.voltages && len(m.Voltages) == 3 {
		data = append(data,
			g.uc.UpdateDataVoltagePhaseA(m.Voltages[0], &now, nil),
			g.uc.UpdateDataVoltagePhaseB(m.Voltages[1], &now, nil),
			g.uc.UpdateDataVoltagePhaseC(m.Voltages[2], &now, nil),
		)
	}

	// an unsupported entry fails the whole update
	supported := data[:0]
	for _, d := range data {
		if d.Supported() {
			supported = append(supported, d)
		}
	}

	if err := g.uc.Update(supported...); err != nil {
		g.log.WARN.Printf("mgcp update: %v", err)
	}
}
