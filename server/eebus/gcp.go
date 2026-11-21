package eebus

import (
	"context"
	"fmt"
	"sync"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	gcpmgcp "github.com/enbility/eebus-go/usecases/gcp/mgcp"
	"github.com/enbility/spine-go/model"
	"github.com/enbility/spine-go/spine"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
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

	mu      sync.Mutex
	last    types.Measurement // last known values, restated with error state while unavailable
	updated time.Time
	stale   bool
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

// update publishes the grid measurement. Values missing from it are restated with error state.
func (g *GridConnectionPoint) update(m types.Measurement) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.updated = time.Now()
	g.stale = false
	g.publish(m, g.last, false)

	if m.Energy == nil {
		m.Energy = g.last.Energy
	}
	if m.ReturnEnergy == nil {
		m.ReturnEnergy = g.last.ReturnEnergy
	}
	if len(m.Currents) != 3 {
		m.Currents = g.last.Currents
	}
	if len(m.Voltages) != 3 {
		m.Voltages = g.last.Voltages
	}
	g.last = m
}

// invalidate restates all values with error state once no measurement arrived within timeout
func (g *GridConnectionPoint) invalidate(timeout time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.stale || g.updated.IsZero() || time.Since(g.updated) < timeout {
		return
	}

	g.stale = true
	g.publish(types.Measurement{}, g.last, true)
}

// Run publishes the grid measurements received from in and invalidates them if the grid meter stops updating
func (g *GridConnectionPoint) Run(ctx context.Context, in <-chan util.Param, timeout time.Duration) {
	tick := time.NewTicker(timeout / 2)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case p := <-in:
			if m, ok := p.Val.(types.Measurement); ok && p.Key == keys.Grid && p.Loadpoint == nil {
				g.update(m)
			}
		case <-tick.C:
			g.invalidate(timeout)
		}
	}
}

// publish writes the values of m, falling back to last with error state for missing values
func (g *GridConnectionPoint) publish(m, last types.Measurement, stale bool) {
	now := time.Now()
	errState := new(model.MeasurementValueStateTypeError)

	state := func(missing bool) *model.MeasurementValueStateType {
		if stale || missing {
			return errState
		}
		return nil
	}

	var data []ucapi.UpdateData

	power := m.Power
	if stale {
		power = last.Power
	}
	data = append(data, g.uc.UpdateDataPowerTotal(power, &now, state(false)))

	// MGCP energy is in Wh
	if v, missing := pick(m.Energy, last.Energy); v != nil {
		data = append(data, g.uc.UpdateDataEnergyConsumed(*v*1e3, &now, state(missing), nil, nil))
	}
	if v, missing := pick(m.ReturnEnergy, last.ReturnEnergy); v != nil {
		data = append(data, g.uc.UpdateDataEnergyFeedIn(*v*1e3, &now, state(missing), nil, nil))
	}

	if v, missing := pickPhases(m.Currents, last.Currents); g.currents && v != nil {
		data = append(data,
			g.uc.UpdateDataCurrentPhaseA(v[0], &now, state(missing)),
			g.uc.UpdateDataCurrentPhaseB(v[1], &now, state(missing)),
			g.uc.UpdateDataCurrentPhaseC(v[2], &now, state(missing)),
		)
	}

	if v, missing := pickPhases(m.Voltages, last.Voltages); g.voltages && v != nil {
		data = append(data,
			g.uc.UpdateDataVoltagePhaseA(v[0], &now, state(missing)),
			g.uc.UpdateDataVoltagePhaseB(v[1], &now, state(missing)),
			g.uc.UpdateDataVoltagePhaseC(v[2], &now, state(missing)),
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

func pick(v, last *float64) (*float64, bool) {
	if v != nil {
		return v, false
	}
	return last, true
}

func pickPhases(v, last []float64) ([]float64, bool) {
	if len(v) == 3 {
		return v, false
	}
	if len(last) == 3 {
		return last, true
	}
	return nil, true
}
