package eebus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/service"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/cert"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	evccapi "github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	server "github.com/evcc-io/evcc/server/eebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ems is a downstream energy manager reading the grid connection point via MGCP
type ems struct {
	controlbox // reuses the service handler callbacks

	mu     sync.Mutex
	entity spineapi.EntityRemoteInterface
	ucmgcp ucapi.MaMGCPInterface
}

func createEMS(ctx context.Context, t *testing.T, remoteSki string) *ems {
	certificate, err := cert.CreateCertificate("Demo", "Demo", "DE", "Demo-EMS-01")
	require.NoError(t, err)

	ski, err := server.SkiFromCert(certificate)
	require.NoError(t, err)

	h := &ems{controlbox: controlbox{ski: ski}}

	configuration, err := api.NewConfiguration(
		"Demo", "Demo", "EMS", ski[:8],
		[]shipapi.DeviceCategoryType{shipapi.DeviceCategoryTypeEnergyManagementSystem},
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{model.EntityTypeTypeCEM},
		freePort(t), certificate, time.Second*4, nil, nil)
	require.NoError(t, err)
	configuration.SetAlternateIdentifier("Demo-EMS-" + ski[:8])

	h.myService = service.NewService(configuration, h)
	require.NoError(t, h.myService.Setup())

	localEntity := h.myService.LocalDevice().EntityForType(model.EntityTypeTypeCEM)
	h.ucmgcp = mgcp.NewMGCP(localEntity, h.onEvent)
	require.NoError(t, h.myService.AddUseCase(h.ucmgcp))

	h.myService.RegisterRemoteService(shipapi.NewServiceIdentity(remoteSki, "", ""))
	h.start(ctx)

	return h
}

func (h *ems) onEvent(_ string, _ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	if event == mgcp.UseCaseSupportUpdate && entity != nil {
		h.mu.Lock()
		h.entity = entity
		h.mu.Unlock()
	}
}

func (h *ems) gcp() spineapi.EntityRemoteInterface {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.entity
}

// gridMeter provides all values MGCP can carry
type gridMeter struct{}

func (gridMeter) CurrentPower() (float64, error)               { return 0, nil }
func (gridMeter) TotalEnergy() (float64, error)                { return 0, nil }
func (gridMeter) ReturnEnergy() (float64, error)               { return 0, nil }
func (gridMeter) Currents() (float64, float64, float64, error) { return 0, 0, 0, nil }
func (gridMeter) Voltages() (float64, float64, float64, error) { return 0, 0, 0, nil }

type device struct{}

func (device) Connect(bool) {}
func (device) UseCaseEvent(spineapi.DeviceRemoteInterface, spineapi.EntityRemoteInterface, api.EventType) {
}

// The grid connection point is announced at runtime and its values reach a downstream EMS.
func TestGridConnectionPoint(t *testing.T) {
	if inst, err := server.Instance(); err == nil {
		inst.Shutdown()
	}

	certificate, err := cert.CreateCertificate("Demo", "Demo", "DE", "Demo-GCP-01")
	require.NoError(t, err)

	public, private, err := server.GetX509KeyPair(certificate)
	require.NoError(t, err)

	_, err = server.NewServer(server.Config{
		Port:        freePort(t),
		Certificate: server.Certificate{Public: public, Private: private},
	})
	require.NoError(t, err)

	inst, err := server.Instance()
	require.NoError(t, err)
	defer inst.Shutdown()

	h := createEMS(t.Context(), t, inst.Ski())
	defer h.myService.Shutdown()

	require.NoError(t, inst.RegisterDevice(h.ski, "", device{}))
	require.Eventually(t, h.isConnected.Load, 30*time.Second, 100*time.Millisecond, "connect")

	var meter interface {
		evccapi.Meter
		evccapi.MeterEnergy
		evccapi.MeterReturnEnergy
		evccapi.PhaseCurrents
		evccapi.PhaseVoltages
	} = gridMeter{}

	gcp, err := inst.AddGridConnectionPoint(meter)
	require.NoError(t, err)
	require.NotNil(t, gcp)

	require.Eventually(t, func() bool { return h.gcp() != nil }, 30*time.Second, 100*time.Millisecond, "mgcp support")
	entity := h.gcp()

	gcp.Update(types.Measurement{
		Power:        -1500,
		Energy:       new(12.5),
		ReturnEnergy: new(3.25),
		Currents:     []float64{-2, -2.5, -2},
		Voltages:     []float64{230, 231, 232},
	})

	require.Eventually(t, func() bool {
		p, err := h.ucmgcp.Power(entity)
		return err == nil && p == -1500
	}, 30*time.Second, 100*time.Millisecond, "power")

	require.Eventually(t, func() bool {
		v, err := h.ucmgcp.VoltagePerPhase(entity)
		return err == nil && len(v) == 3
	}, 10*time.Second, 100*time.Millisecond, "voltages")

	energy, err := h.ucmgcp.EnergyConsumed(entity)
	require.NoError(t, err)
	assert.Equal(t, 12500.0, energy)

	feedIn, err := h.ucmgcp.EnergyFeedIn(entity)
	require.NoError(t, err)
	assert.Equal(t, 3250.0, feedIn)

	currents, err := h.ucmgcp.CurrentPerPhase(entity)
	require.NoError(t, err)
	assert.Equal(t, []float64{-2, -2.5, -2}, currents)

	voltages, err := h.ucmgcp.VoltagePerPhase(entity)
	require.NoError(t, err)
	assert.Equal(t, []float64{230, 231, 232}, voltages)
}
