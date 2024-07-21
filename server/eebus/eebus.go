package eebus

import (
	"crypto/tls"
	"errors"
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
	"github.com/enbility/eebus-go/usecases/cs/lpc"
	"github.com/enbility/eebus-go/usecases/cs/lpp"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	shipapi "github.com/enbility/ship-go/api"
	shiputil "github.com/enbility/ship-go/util"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
)

const (
	BrandName  string = "EVCC"
	Model      string = "HEMS"
	DeviceCode string = "EVCC_HEMS_01" // used as common name in cert generation
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
type UseCasesCS struct {
	LPC  ucapi.CsLPCInterface
	LPP  ucapi.CsLPPInterface
	MGCP ucapi.MaMGCPInterface
}

type EEBus struct {
	service eebusapi.ServiceInterface

	evseUC UseCasesEVSE
	csUC   UseCasesCS

	mux sync.Mutex
	log *util.Logger

	SKI string

	clients map[string][]EEBUSDeviceInterface
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
		BrandName, BrandName, Model, serial,
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{model.EntityTypeTypeCEM},
		port, certificate, time.Second*4,
	)
	if err != nil {
		return nil, err
	}

	// for backward compatibility
	configuration.SetAlternateMdnsServiceName(DeviceCode)
	configuration.SetAlternateIdentifier(serial)
	configuration.SetInterfaces(cc.Interfaces)

	ski, err := SkiFromCert(certificate)
	if err != nil {
		return nil, err
	}

	c := &EEBus{
		log:     log,
		SKI:     ski,
		clients: make(map[string][]EEBUSDeviceInterface),
	}

	c.service = service.NewService(configuration, c)
	c.service.SetLogging(c)
	if err := c.service.Setup(); err != nil {
		return nil, err
	}

	localEntity := c.service.LocalDevice().EntityForType(model.EntityTypeTypeCEM)

	// evse
	c.evseUC = UseCasesEVSE{
		EvseCC: evsecc.NewEVSECC(localEntity, c.ucCallback),
		EvCC:   evcc.NewEVCC(c.service, localEntity, c.ucCallback),
		EvCem:  evcem.NewEVCEM(c.service, localEntity, c.ucCallback),
		OpEV:   opev.NewOPEV(localEntity, c.ucCallback),
		OscEV:  oscev.NewOSCEV(localEntity, c.ucCallback),
		EvSoc:  evsoc.NewEVSOC(localEntity, c.ucCallback),
	}

	// controllable system
	c.csUC = UseCasesCS{
		LPC:  lpc.NewLPC(localEntity, c.ucCallback),
		LPP:  lpp.NewLPP(localEntity, c.ucCallback),
		MGCP: mgcp.NewMGCP(localEntity, c.ucCallback),
	}

	// register use cases
	for _, uc := range []eebusapi.UseCaseInterface{
		c.evseUC.EvseCC, c.evseUC.EvCC,
		c.evseUC.EvCem, c.evseUC.OpEV,
		c.evseUC.OscEV, c.evseUC.EvSoc,
		c.csUC.LPC, c.csUC.LPP, c.csUC.MGCP,
	} {
		c.service.AddUseCase(uc)
	}

	return c, nil
}

func (c *EEBus) RegisterDevice(ski string, device EEBUSDeviceInterface) error {
	ski = shiputil.NormalizeSKI(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.SKI {
		return errors.New("device ski can not be identical to host ski")
	}

	c.service.RegisterRemoteSKI(ski)

	c.mux.Lock()
	defer c.mux.Unlock()
	c.clients[ski] = append(c.clients[ski], device)

	return nil
}

func (c *EEBus) Evse() *UseCasesEVSE {
	return &c.evseUC
}

func (c *EEBus) ControllableSystem() *UseCasesCS {
	return &c.csUC
}

func (c *EEBus) Run() {
	c.service.Start()
}

func (c *EEBus) Shutdown() {
	c.service.Shutdown()
}

// Use case callback
func (c *EEBus) ucCallback(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if clients, ok := c.clients[ski]; ok {
		for _, client := range clients {
			client.UseCaseEventCB(device, entity, event)
		}
	}
}

// EEBUSServiceHandler

func (c *EEBus) RemoteSKIConnected(service eebusapi.ServiceInterface, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if clients, ok := c.clients[ski]; ok {
		for _, client := range clients {
			client.DeviceConnect()
		}
	}
}

func (c *EEBus) RemoteSKIDisconnected(service eebusapi.ServiceInterface, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if clients, ok := c.clients[ski]; ok {
		for _, client := range clients {
			client.DeviceConnect()
		}
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
