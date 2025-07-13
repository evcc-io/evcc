package eebus

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"slices"
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
	"github.com/enbility/ship-go/mdns"
	shiputil "github.com/enbility/ship-go/util"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
)

type Device interface {
	Connect(connected bool)
	UseCaseEvent(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType)
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

	clients map[string][]Device
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

	protectedID := machine.ProtectedID("evcc-eebus")
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

	// use avahi if available, otherwise use go based zeroconf
	configuration.SetMdnsProviderSelection(mdns.MdnsProviderSelectionAll)

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
		clients: make(map[string][]Device),
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

func (c *EEBus) RegisterDevice(ski, ip string, device Device) error {
	ski = shiputil.NormalizeSKI(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.SKI {
		return errors.New("device ski can not be identical to host ski")
	}

	if len(ip) > 0 {
		c.service.RemoteServiceForSKI(ski).SetIPv4(ip)
	}
	c.service.RegisterRemoteSKI(ski)

	c.mux.Lock()
	defer c.mux.Unlock()
	c.clients[ski] = append(c.clients[ski], device)

	return nil
}

func (c *EEBus) UnregisterDevice(ski string, device Device) {
	ski = shiputil.NormalizeSKI(ski)
	c.log.TRACE.Printf("unregistering ski: %s", ski)

	c.service.UnregisterRemoteSKI(ski)

	c.mux.Lock()
	defer c.mux.Unlock()

	if idx := slices.Index(c.clients[ski], device); idx != -1 {
		c.clients[ski] = slices.Delete(c.clients[ski], idx, idx+1)
	}
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

	c.log.DEBUG.Printf("ski %s event %s", ski, event)

	if clients, ok := c.clients[ski]; ok {
		for _, client := range clients {
			client.UseCaseEvent(device, entity, event)
		}
	}
}

// EEBUSServiceHandler

func (c *EEBus) connect(ski string, connected bool) {
	action := map[bool]string{true: "connected", false: "disconnected"}[connected]
	c.log.DEBUG.Printf("ski %s %s", ski, action)

	c.mux.Lock()
	defer c.mux.Unlock()

	if clients, ok := c.clients[ski]; ok {
		for _, client := range clients {
			client.Connect(connected)
		}
	}
}

func (c *EEBus) RemoteSKIConnected(service eebusapi.ServiceInterface, ski string) {
	c.connect(ski, true)
}

func (c *EEBus) RemoteSKIDisconnected(service eebusapi.ServiceInterface, ski string) {
	c.connect(ski, false)
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

	if clients, ok := c.clients[ski]; !ok || len(clients) == 0 {
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
