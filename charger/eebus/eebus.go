package eebus

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"dario.cat/mergo"
	eebusapi "github.com/enbility/eebus-go/api"
	service "github.com/enbility/eebus-go/service"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/evcc"
	"github.com/enbility/eebus-go/usecases/cem/evcem"
	"github.com/enbility/eebus-go/usecases/cem/evsecc"
	"github.com/enbility/eebus-go/usecases/cem/evsoc"
	"github.com/enbility/eebus-go/usecases/cem/opev"
	"github.com/enbility/eebus-go/usecases/cem/oscev"
	shipapi "github.com/enbility/ship-go/api"
	shiputil "github.com/enbility/ship-go/util"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
)

const (
	EEBUSBrandName  string = "EVCC"
	EEBUSModel      string = "HEMS"
	EEBUSDeviceCode string = "EVCC_HEMS_01" // used as common name in cert generation
)

type Config struct {
	URI         string
	ShipID      string
	Interfaces  []string
	Certificate struct {
		Public, Private string
	}
}

// Configured returns true if the EEbus server is configured
func (c Config) Configured() bool {
	return len(c.Certificate.Public) > 0 && len(c.Certificate.Private) > 0
}

type EEBUSDeviceInterface interface {
	DeviceConnect()
	DeviceDisconnect()
	UseCaseEventCB(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType)
}

// EVSE UseCases
type UseCasesEVSE struct {
	EvseCC ucapi.CemEVSECCInterface
	EvCC   ucapi.CemEVCCInterface
	EvCem  ucapi.CemEVCEMInterface
	EvSoc  ucapi.CemEVSOCInterface
	OpEV   ucapi.CemOPEVInterface
	OscEV  ucapi.CemOSCEVInterface
}

type EEBus struct {
	service eebusapi.ServiceInterface

	evseUC *UseCasesEVSE

	mux sync.Mutex
	log *util.Logger

	SKI string

	clients map[string]EEBUSDeviceInterface
}

var Instance *EEBus

func NewServer(other Config) (*EEBus, error) {
	cc := Config{
		URI: ":4712",
	}

	if err := mergo.Merge(&cc, other, mergo.WithOverride); err != nil {
		return nil, err
	}

	log := util.NewLogger("eebus")

	protectedID, err := machine.ProtectedID("evcc-eebus")
	if err != nil {
		return nil, err
	}
	serial := fmt.Sprintf("%s-%0x", "EVCC", protectedID[:8])

	if len(cc.ShipID) != 0 {
		serial = cc.ShipID
	}

	certificate, err := tls.X509KeyPair([]byte(cc.Certificate.Public), []byte(cc.Certificate.Private))
	if err != nil {
		return nil, err
	}

	_, portValue, err := net.SplitHostPort(cc.URI)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return nil, err
	}

	// TODO: get the voltage from the site
	configuration, err := eebusapi.NewConfiguration(
		EEBUSBrandName, EEBUSBrandName, EEBUSModel, serial,
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{model.EntityTypeTypeCEM},
		port, certificate, time.Second*4,
	)
	if err != nil {
		return nil, err
	}

	// for backward compatibility
	configuration.SetAlternateMdnsServiceName(EEBUSDeviceCode)
	configuration.SetAlternateIdentifier(serial)
	configuration.SetInterfaces(cc.Interfaces)

	ski, err := SkiFromCert(certificate)
	if err != nil {
		return nil, err
	}

	c := &EEBus{
		log:     log,
		clients: make(map[string]EEBUSDeviceInterface),
		SKI:     ski,
	}

	c.service = service.NewService(configuration, c)
	c.service.SetLogging(c)
	if err := c.service.Setup(); err != nil {
		return nil, err
	}

	localEntity := c.service.LocalDevice().EntityForType(model.EntityTypeTypeCEM)

	c.evseUC = &UseCasesEVSE{
		EvseCC: evsecc.NewEVSECC(localEntity, c.evseUsecaseCB),
		EvCC:   evcc.NewEVCC(c.service, localEntity, c.evseUsecaseCB),
		EvCem:  evcem.NewEVCEM(c.service, localEntity, c.evseUsecaseCB),
		OpEV:   opev.NewOPEV(localEntity, c.evseUsecaseCB),
		OscEV:  oscev.NewOSCEV(localEntity, c.evseUsecaseCB),
		EvSoc:  evsoc.NewEVSOC(localEntity, c.evseUsecaseCB),
	}

	// register use cases
	for _, uc := range []eebusapi.UseCaseInterface{
		c.evseUC.EvseCC, c.evseUC.EvCC,
		c.evseUC.EvCem, c.evseUC.OpEV,
		c.evseUC.OscEV, c.evseUC.EvSoc,
	} {
		c.service.AddUseCase(uc)
	}

	return c, nil
}

func (c *EEBus) RegisterEVSE(ski string, device EEBUSDeviceInterface) *UseCasesEVSE {
	ski = shiputil.NormalizeSKI(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.SKI {
		c.log.FATAL.Fatal("The charger SKI can not be identical to the SKI of evcc!")
	}

	c.service.RegisterRemoteSKI(ski)

	c.mux.Lock()
	defer c.mux.Unlock()
	c.clients[ski] = device

	return c.evseUC
}

func (c *EEBus) Run() {
	c.service.Start()
}

func (c *EEBus) Shutdown() {
	c.service.Shutdown()
}

// EVSE/EV UseCase CB
func (c *EEBus) evseUsecaseCB(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if client, ok := c.clients[ski]; ok {
		client.UseCaseEventCB(device, entity, event)
	}
}

// EEBUSServiceHandler

func (c *EEBus) RemoteSKIConnected(service eebusapi.ServiceInterface, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if client, ok := c.clients[ski]; ok {
		client.DeviceConnect()
	}
}

func (c *EEBus) RemoteSKIDisconnected(service eebusapi.ServiceInterface, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if client, ok := c.clients[ski]; ok {
		client.DeviceConnect()
	}
}

// report all currently visible EEBUS services
// this is needed to provide an UI for pairing with other devices
// if not all incoming pairing requests should be accepted
func (c *EEBus) VisibleRemoteServicesUpdated(service eebusapi.ServiceInterface, entries []shipapi.RemoteService) {
}

// Provides the SHIP ID the remote service reported during the handshake process
// This needs to be persisted and passed on for future remote service connections
// when using `PairRemoteService`
func (c *EEBus) ServiceShipIDUpdate(ski string, shipdID string) {}

// Provides the current pairing state for the remote service
// This is called whenever the state changes and can be used to
// provide user information for the pairing/connection process
func (c *EEBus) ServicePairingDetailUpdate(ski string, detail *shipapi.ConnectionStateDetail) {
	if detail.State() != shipapi.ConnectionStateReceivedPairingRequest {
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	if _, ok := c.clients[ski]; !ok {
		// this is an unknown SKI, so deny pairing
		c.service.CancelPairingWithSKI(ski)
	}
}

// EEBUS Logging interface

func (c *EEBus) Trace(args ...interface{}) {
	c.log.TRACE.Println(args...)
}

func (c *EEBus) Tracef(format string, args ...interface{}) {
	c.log.TRACE.Printf(format, args...)
}

func (c *EEBus) Debug(args ...interface{}) {
	c.log.DEBUG.Println(args...)
}

func (c *EEBus) Debugf(format string, args ...interface{}) {
	c.log.DEBUG.Printf(format, args...)
}

func (c *EEBus) Info(args ...interface{}) {
	c.log.INFO.Println(args...)
}

func (c *EEBus) Infof(format string, args ...interface{}) {
	c.log.INFO.Printf(format, args...)
}

func (c *EEBus) Error(args ...interface{}) {
	c.log.ERROR.Println(args...)
}

func (c *EEBus) Errorf(format string, args ...interface{}) {
	c.log.ERROR.Printf(format, args...)
}
