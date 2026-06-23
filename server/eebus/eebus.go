package eebus

import (
	"crypto/tls"
	"errors"
	"fmt"
	"slices"
	"strings"
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
	csplc "github.com/enbility/eebus-go/usecases/cs/lpc"
	cslpp "github.com/enbility/eebus-go/usecases/cs/lpp"
	eglpc "github.com/enbility/eebus-go/usecases/eg/lpc"
	eglpp "github.com/enbility/eebus-go/usecases/eg/lpp"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/mdns"
	shiputil "github.com/enbility/ship-go/util"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/enbility/spine-go/spine"
	"github.com/evcc-io/evcc/util"
)

type Device interface {
	Connect(connected bool)
	UseCaseEvent(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType)
}

// Customer Energy Management
type CustomerEnergyManagement struct {
	EvseCC ucapi.CemEVSECCInterface
	EvCC   ucapi.CemEVCCInterface
	EvCem  ucapi.CemEVCEMInterface
	EvSoc  ucapi.CemEVSOCInterface
	OpEV   ucapi.CemOPEVInterface
	OscEV  ucapi.CemOSCEVInterface
}

// Controllable System
type ControllableSystem struct {
	ucapi.CsLPCInterface
	ucapi.CsLPPInterface
}

// Monitoring Appliance
type MonitoringAppliance struct {
	ucapi.MaMGCPInterface
	ucapi.MaMPCInterface
}

// Energy Guard
type EnergyGuard struct {
	ucapi.EgLPCInterface
	ucapi.EgLPPInterface
}

type EEBus struct {
	service        eebusapi.ServiceInterface
	remoteServices []shipapi.RemoteMdnsService

	cem CustomerEnergyManagement
	cs  ControllableSystem
	ma  MonitoringAppliance
	eg  EnergyGuard

	mux sync.Mutex
	log *util.Logger

	ski string

	clients map[string][]Device
}

var Instance *EEBus

func GetStatus() any {
	return struct {
		Ski string `json:"ski"`
		QR  string `json:"qr,omitempty"`
	}{
		Ski: Ski(),
		QR:  qrCode(),
	}
}

// qrCode returns the SHIP installation QR code text per EEBus SHIP installation
// requirements, or empty if unavailable
func qrCode() string {
	if Instance == nil {
		return ""
	}
	qr, err := Instance.service.QRCodeText()
	if err != nil {
		return ""
	}
	return qr
}

func NewServer(other Config) (*EEBus, error) {
	cc := Config{
		Port: 4712,
	}

	if err := mergo.Merge(&cc, other, mergo.WithOverride); err != nil {
		return nil, err
	}

	serial := cc.ShipID
	if serial == "" {
		serial = createShipID()
	}

	certificate, err := tls.X509KeyPair([]byte(cc.Certificate.Public), []byte(cc.Certificate.Private))
	if err != nil {
		return nil, err
	}

	configuration, err := eebusapi.NewConfiguration(
		BrandName, BrandName, Model, serial,
		[]shipapi.DeviceCategoryType{shipapi.DeviceCategoryTypeEnergyManagementSystem},
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{model.EntityTypeTypeCEM},
		cc.Port, certificate, time.Second*4,
		// no SHIP Pairing (and thus no ring buffer persistence): remote services are
		// trusted by their configured SKI via RegisterRemoteService
		nil, nil,
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
		log:     util.NewLogger("eebus"),
		ski:     ski,
		clients: make(map[string][]Device),
	}

	c.service = service.NewService(configuration, c)
	c.service.SetLogging(c)
	if err := c.service.Setup(); err != nil {
		return nil, err
	}

	localDevice := c.service.LocalDevice()

	{
		// CEM entity for for connected EVSE and Meters
		localEntity := localDevice.Entity([]model.AddressEntityType{1})

		// customer energy management to EVSE
		c.cem = CustomerEnergyManagement{
			EvseCC: evsecc.NewEVSECC(localEntity, c.ucCallback),
			EvCC:   evcc.NewEVCC(c.service, localEntity, c.ucCallback),
			EvCem:  evcem.NewEVCEM(c.service, localEntity, c.ucCallback),
			OpEV:   opev.NewOPEV(localEntity, c.ucCallback),
			OscEV:  oscev.NewOSCEV(localEntity, c.ucCallback),
			EvSoc:  evsoc.NewEVSOC(localEntity, c.ucCallback),
		}

		// monitoring appliance to meters
		c.ma = MonitoringAppliance{
			MaMGCPInterface: mgcp.NewMGCP(localEntity, c.ucCallback),
			MaMPCInterface:  mpc.NewMPC(localEntity, c.ucCallback),
		}
	}

	{
		// CEM entity for connected SMGW
		// LPC/LPP use a 60s heartbeat timeout, but some EVSE devices have then issues when not set to 4s right now even though they should not connect to this one anyway
		localEntity := spine.NewEntityLocal(localDevice, model.EntityTypeTypeCEM, []model.AddressEntityType{2}, time.Second*4)
		localDevice.AddEntity(localEntity)

		// controllable system
		c.cs = ControllableSystem{
			CsLPCInterface: csplc.NewLPC(localEntity, c.ucCallback),
			CsLPPInterface: cslpp.NewLPP(localEntity, c.ucCallback),
		}
	}

	{
		// GridGuard entity for connected Controllable Systems
		// LPC/LPP use a 60s heartbeat timeout, but some EVSE devices have then issues when not set to 4s right now
		localEntity := spine.NewEntityLocal(localDevice, model.EntityTypeTypeGridGuard, []model.AddressEntityType{3}, time.Second*4)
		localDevice.AddEntity(localEntity)

		// energy guard
		c.eg = EnergyGuard{
			EgLPCInterface: eglpc.NewLPC(localEntity, c.ucCallback),
			EgLPPInterface: eglpp.NewLPP(localEntity, c.ucCallback),
		}
	}

	// register use cases
	for _, uc := range []eebusapi.UseCaseInterface{
		c.cem.EvseCC, c.cem.EvCC,
		c.cem.EvCem, c.cem.OpEV,
		c.cem.OscEV, c.cem.EvSoc,
		c.cs.CsLPCInterface, c.cs.CsLPPInterface,
		c.ma.MaMGCPInterface, c.ma.MaMPCInterface,
		c.eg.EgLPCInterface, c.eg.EgLPPInterface,
	} {
		c.service.AddUseCase(uc)
	}

	return c, nil
}

func (c *EEBus) RegisterDevice(ski, ip string, device Device) error {
	ski = shiputil.NormalizeSKI(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.ski {
		return errors.New("device ski can not be identical to host ski")
	}

	identity := shipapi.NewServiceIdentity(ski, "", "")
	if len(ip) > 0 {
		identity.IPv4 = ip
	}
	c.service.RegisterRemoteService(identity)

	c.mux.Lock()
	defer c.mux.Unlock()
	c.clients[ski] = append(c.clients[ski], device)

	return nil
}

func (c *EEBus) UnregisterDevice(ski string, device Device) {
	ski = shiputil.NormalizeSKI(ski)
	c.log.TRACE.Printf("unregistering ski: %s", ski)

	c.mux.Lock()
	if idx := slices.Index(c.clients[ski], device); idx != -1 {
		c.clients[ski] = slices.Delete(c.clients[ski], idx, idx+1)

		if len(c.clients[ski]) == 0 {
			delete(c.clients, ski)

			// Tear down the SHIP session after releasing the mutex: ship-go's CloseConnection
			// on a non-Complete state synchronously invokes HandleConnectionClosed,
			// which calls back into evcc's connect(ski, false) — and that needs to
			// acquire c.mux. Holding c.mux across this cross-layer call would
			// deadlock the same goroutine on its own non-reentrant mutex. See #28942.
			defer c.service.UnregisterRemoteService(shipapi.NewServiceIdentity(ski, "", ""))
		}
	}
	c.mux.Unlock()
}

func (c *EEBus) CustomerEnergyManagement() *CustomerEnergyManagement {
	return &c.cem
}

func (c *EEBus) ControllableSystem() *ControllableSystem {
	return &c.cs
}

func (c *EEBus) MonitoringAppliance() *MonitoringAppliance {
	return &c.ma
}

func (c *EEBus) EnergyGuard() *EnergyGuard {
	return &c.eg
}

func (c *EEBus) RemoteServices() []shipapi.RemoteMdnsService {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.remoteServices
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

func (c *EEBus) RemoteServiceConnected(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity) {
	c.connect(identity.SKI, true)
}

func (c *EEBus) RemoteServiceDisconnected(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity) {
	c.connect(identity.SKI, false)
}

// report all currently visible EEBUS services
// this is needed to provide an UI for pairing with other devices
// if not all incoming pairing requests should be accepted
func (c *EEBus) VisibleRemoteMdnsServicesUpdated(service eebusapi.ServiceInterface, entries []shipapi.RemoteMdnsService) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.remoteServices = slices.Clone(entries)
}

// Provides updated service information (ShipID, fingerprint, ...) discovered
// during the handshake process. This needs to be persisted and passed on for
// future remote service connections when using `RegisterRemoteService`
func (c *EEBus) ServiceUpdated(identity shipapi.ServiceIdentity) {}

// Provides the current pairing state for the remote service
// This is called whenever the state changes and can be used to
// provide user information for the pairing/connection process
func (c *EEBus) ServicePairingDetailUpdate(identity shipapi.ServiceIdentity, detail *shipapi.ConnectionStateDetail) {
	if detail.State() != shipapi.ConnectionStateReceivedPairingRequest {
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	if clients, ok := c.clients[identity.SKI]; !ok || len(clients) == 0 {
		// this is an unknown SKI, so deny pairing
		c.service.CancelPairing(identity)
	}
}

// SHIP Pairing Service events: evcc trusts remote services by configured SKI via
// RegisterRemoteService and does not use SHIP Pairing; logged for visibility only
func (c *EEBus) ServiceAutoTrusted(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity) {
	c.log.INFO.Printf("service trusted: %s", identity.SKI)
}

func (c *EEBus) ServiceAutoTrustFailed(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity, reason error) {
	c.log.INFO.Printf("service trust failed: %s: %v", identity.SKI, reason)
}

func (c *EEBus) ServiceAutoTrustRemoved(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity, reason string) {
	c.log.INFO.Printf("service trust removed: %s: %s", identity.SKI, reason)
}

// EEBUS Logging interface

func (c *EEBus) Trace(args ...any) {
	c.log.TRACE.Println(args...)
}

func (c *EEBus) Tracef(format string, args ...any) {
	c.log.TRACE.Printf(format, args...)
}

func isRelevant(s string) bool {
	return strings.Contains(s, "connect") || strings.Contains(s, " event ")
}

func (c *EEBus) Debug(args ...any) {
	if s := fmt.Sprint(args...); isRelevant(s) {
		c.log.DEBUG.Print(s)
	}
}

func (c *EEBus) Debugf(format string, args ...any) {
	if s := fmt.Sprintf(format, args...); isRelevant(s) {
		c.log.DEBUG.Print(s)
	}
}

func (c *EEBus) Info(args ...any) {
	c.log.INFO.Println(args...)
}

func (c *EEBus) Infof(format string, args ...any) {
	c.log.INFO.Printf(format, args...)
}

func (c *EEBus) Error(args ...any) {
	if len(args) == 2 {
		// TODO remove when enbility/ship-go is upgraded
		if err := fmt.Sprint(args...); err == "websocket server error:http: Server closed" {
			return
		}
	}

	c.log.ERROR.Println(args...)
}

func (c *EEBus) Errorf(format string, args ...any) {
	c.log.ERROR.Printf(format, args...)
}
