package eebus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/service"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/eebus-go/usecases/eg/lpp"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/cert"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	server "github.com/evcc-io/evcc/server/eebus"
)

type controlbox struct {
	mu sync.Mutex

	ski       string
	myService *service.Service

	uclpc ucapi.EgLPCInterface
	uclpp ucapi.EgLPPInterface

	remoteEntities map[api.EventType][]spineapi.EntityRemoteInterface
	remoteEventC   chan<- api.EventType

	isConnected bool
}

func createControlbox(ctx context.Context, remoteSki string, port int) (*controlbox, error) {
	h, err := newControlbox(nil, port)
	if err != nil {
		return nil, err
	}

	h.myService.RegisterRemoteService(shipapi.NewServiceIdentity(remoteSki, "", ""))
	h.start(ctx)

	return h, nil
}

func newControlbox(pairing *shipapi.PairingConfig, port int) (*controlbox, error) {
	certificate, err := cert.CreateCertificate("Demo", "Demo", "DE", "Demo-Unit-01")
	if err != nil {
		return nil, err
	}

	ski, err := server.SkiFromCert(certificate)
	if err != nil {
		return nil, err
	}

	h := &controlbox{
		ski: ski,
	}

	// unique per instance: a shared serial collides on ShipID when multiple controlboxes run concurrently
	serial := ski[:8]

	configuration, err := api.NewConfiguration(
		"Demo", "Demo", "ControlBox", serial,
		[]shipapi.DeviceCategoryType{shipapi.DeviceCategoryTypeGridConnectionHub},
		model.DeviceTypeTypeElectricitySupplySystem,
		[]model.EntityTypeType{model.EntityTypeTypeGridGuard},
		port, certificate, time.Second*60, pairing, nil)
	if err != nil {
		return nil, err
	}
	configuration.SetAlternateIdentifier("Demo-ControlBox-" + serial)

	h.myService = service.NewService(configuration, h)
	// h.myService.SetLogging(h)

	if err = h.myService.Setup(); err != nil {
		return nil, err
	}

	localEntity := h.myService.LocalDevice().EntityForType(model.EntityTypeTypeGridGuard)
	h.uclpc = lpc.NewLPC(localEntity, h.OnLPCEvent)
	h.myService.AddUseCase(h.uclpc)

	h.uclpp = lpp.NewLPP(localEntity, h.OnLPPEvent)
	h.myService.AddUseCase(h.uclpp)

	return h, nil
}

func (h *controlbox) start(ctx context.Context) {
	h.myService.Start()

	go func() {
		<-ctx.Done()
		h.myService.Shutdown()
	}()
}

func (h *controlbox) remoteEntity(event api.EventType) []spineapi.EntityRemoteInterface {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.remoteEntities[event]
}

func (h *controlbox) registerRemoteEntity(entity spineapi.EntityRemoteInterface, event api.EventType) {
	h.mu.Lock()
	defer h.mu.Unlock()

	defer func() {
		if h.remoteEventC != nil {
			h.remoteEventC <- event
		}
	}()

	if h.remoteEntities == nil {
		h.remoteEntities = make(map[api.EventType][]spineapi.EntityRemoteInterface)
	}

	h.remoteEntities[event] = append(h.remoteEntities[event], entity)
}

// LPC
func (h *controlbox) OnLPCEvent(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	if !h.isConnected {
		return
	}

	switch event {
	case lpc.UseCaseSupportUpdate:
		h.registerRemoteEntity(entity, event)
		if currentLimit, err := h.uclpc.ConsumptionLimit(entity); err == nil {
			fmt.Println("New consumption limit received", currentLimit.Value, "W")
		}

	case lpc.DataUpdateLimit:
		h.registerRemoteEntity(entity, event)
	default:
		fmt.Println("lpc:", event)
	}
}

// LPP
func (h *controlbox) OnLPPEvent(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	if !h.isConnected {
		return
	}

	switch event {
	case lpp.UseCaseSupportUpdate:
		h.registerRemoteEntity(entity, event)
	// case lpp.DataUpdateLimit:
	// 	if currentLimit, err := h.uclpp.ConsumptionLimit(entity); err == nil {
	// 		fmt.Println("New consumption limit received", currentLimit.Value, "W")
	// 	}
	default:
		fmt.Println("lpp:", event)
	}
}

// EEBUSServiceHandler

func (h *controlbox) RemoteServiceConnected(service api.ServiceInterface, identity shipapi.ServiceIdentity) {
	h.isConnected = true
}

func (h *controlbox) RemoteServiceDisconnected(service api.ServiceInterface, identity shipapi.ServiceIdentity) {
	h.isConnected = false
}

func (h *controlbox) VisibleRemoteMdnsServicesUpdated(service api.ServiceInterface, entries []shipapi.RemoteMdnsService) {
}

func (h *controlbox) ServiceUpdated(identity shipapi.ServiceIdentity) {
}

func (h *controlbox) ServicePairingDetailUpdate(identity shipapi.ServiceIdentity, detail *shipapi.ConnectionStateDetail) {
}

func (h *controlbox) ServiceAutoTrusted(service api.ServiceInterface, identity shipapi.ServiceIdentity) {
}

func (h *controlbox) ServiceAutoTrustFailed(service api.ServiceInterface, identity shipapi.ServiceIdentity, reason error) {
}

func (h *controlbox) ServiceAutoTrustRemoved(service api.ServiceInterface, identity shipapi.ServiceIdentity, reason string) {
}

func (h *controlbox) AllowWaitingForTrust(identity shipapi.ServiceIdentity) bool {
	return true
}
