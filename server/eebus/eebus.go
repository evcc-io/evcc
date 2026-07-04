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
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
	"github.com/enbility/eebus-go/usecases/cem/opev"
	"github.com/enbility/eebus-go/usecases/cem/oscev"
	csplc "github.com/enbility/eebus-go/usecases/cs/lpc"
	cslpp "github.com/enbility/eebus-go/usecases/cs/lpp"
	eglpc "github.com/enbility/eebus-go/usecases/eg/lpc"
	eglpp "github.com/enbility/eebus-go/usecases/eg/lpp"
	"github.com/enbility/eebus-go/usecases/ma/mdt"
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
	OHPCF  ucapi.CemOHPCFInterface
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
	ucapi.MaMDTInterface
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

	paired    []shipapi.ServiceIdentity // devices paired via SHIP Pairing Service
	connected map[string]bool           // connection state per ski

	clients map[string][]Device
}

var (
	instance *EEBus
	started  func() error // memoized service start; set in NewServer, runs once
)

// Instance returns the eebus server, starting the service once on first call
// (OCPP pattern). Returns an error if eebus is unconfigured or start fails.
func Instance() (*EEBus, error) {
	if instance == nil {
		return nil, errors.New("eebus not configured")
	}
	if err := started(); err != nil {
		return nil, err
	}
	return instance, nil
}

func GetStatus() any {
	var ski string
	if instance != nil {
		ski = instance.Ski()
	}
	return struct {
		Ski string `json:"ski"`
		QR  string `json:"qr,omitempty"`
	}{
		Ski: ski,
		QR:  qrCode(),
	}
}

// qrCode returns the SHIP installation QR code text per EEBus SHIP installation
// requirements, or empty if unavailable
func qrCode() string {
	if instance == nil {
		return ""
	}
	qr, err := instance.service.QRCodeText()
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

	pairingConfig, ringBuffer, err := pairing(cc.Secret)
	if err != nil {
		return nil, err
	}

	configuration, err := eebusapi.NewConfiguration(
		BrandName, BrandName, Model, serial,
		[]shipapi.DeviceCategoryType{shipapi.DeviceCategoryTypeEnergyManagementSystem},
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{model.EntityTypeTypeCEM},
		cc.Port, certificate, time.Second*4,
		pairingConfig, ringBuffer,
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
		log:       util.NewLogger("eebus"),
		ski:       ski,
		clients:   make(map[string][]Device),
		connected: make(map[string]bool),
	}

	c.service = service.NewService(configuration, c)
	c.service.SetLogging(c)
	if err := c.service.Setup(); err != nil {
		if errors.Is(err, shipapi.ErrInvalidSKI) {
			const hint = "The stored EEBUS certificate has an invalid Subject Key Identifier (SKI).\n" +
				"The most common cause is a multi-year-old certificate whose SKI format is no longer accepted\n" +
				"by the stricter validation introduced in evcc 0.309.2 — see\n" +
				"https://github.com/evcc-io/evcc/issues/31366 for context.\n" +
				"To fix this, delete the EEBUS configuration and generate a new certificate:\n" +
				"  1. Open the evcc UI at Configuration > Services > EEBUS and remove the existing configuration, or\n" +
				"     delete the EEBUS section from your evcc.yaml.\n" +
				"  2. Generate a new certificate via the UI, or follow https://docs.evcc.io/de/reference/configuration/eebus/ for evcc.yaml.\n" +
				"  3. Re-pair each EEBUS device (wallbox, heat pump, etc.) with the new SKI.\n" +
				"§14a-EnWG users: do NOT run steps 1–3 on your production system yet. Stay on evcc < 0.309.2\n" +
				"with the old certificate; generate a new one on the side only to send its SKI to your\n" +
				"metering point operator for acceptance (this can take weeks). See\n" +
				"https://docs.evcc.io/de/features/external-control/ for the §14a feature."
			return nil, fmt.Errorf("%w\n\n%s", err, hint)
		}
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
			OHPCF:  ohpcf.NewOHPCF(localEntity, c.ucCallback),
		}

		// monitoring appliance to meters
		c.ma = MonitoringAppliance{
			MaMGCPInterface: mgcp.NewMGCP(localEntity, c.ucCallback),
			MaMPCInterface:  mpc.NewMPC(localEntity, c.ucCallback),
			MaMDTInterface:  mdt.NewMDT(localEntity, c.ucCallback),
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
		c.cem.OHPCF,
		c.cs.CsLPCInterface, c.cs.CsLPPInterface,
		c.ma.MaMGCPInterface, c.ma.MaMPCInterface, c.ma.MaMDTInterface,
		c.eg.EgLPCInterface, c.eg.EgLPPInterface,
	} {
		c.service.AddUseCase(uc)
	}

	// re-establish trust for devices paired via the SHIP Pairing Service
	if pairingConfig != nil {
		identities, err := trustedDevices()
		if err != nil {
			c.log.ERROR.Printf("loading paired devices: %v", err)
		}
		c.paired = identities
		for _, identity := range c.paired {
			c.service.RegisterRemoteService(identity)
		}
	}

	started = sync.OnceValue(c.service.Start)
	instance = c

	return c, nil
}

// Ski returns the local service SKI.
func (c *EEBus) Ski() string {
	return c.ski
}

// PairingSource identifies how trust for a device was established
type PairingSource string

const (
	PairingSourcePaired PairingSource = "paired" // trusted via SHIP Pairing Service, removable
	PairingSourceSki    PairingSource = "ski"    // trusted by configured SKI, not removable here
)

// PairingInfo describes a trusted device, regardless of how trust was established
type PairingInfo struct {
	shipapi.ServiceIdentity
	Source PairingSource `json:"source"`
}

// Pairings returns all trusted devices, tagged by how trust was established
func (c *EEBus) Pairings() []PairingInfo {
	c.mux.Lock()
	defer c.mux.Unlock()

	res := make([]PairingInfo, 0, len(c.paired)+len(c.clients))
	for _, identity := range c.paired {
		res = append(res, PairingInfo{ServiceIdentity: identity, Source: PairingSourcePaired})
	}

	for ski := range c.clients {
		if ski == "" || c.pairedIndex(shipapi.NewServiceIdentity(ski, "", "")) >= 0 {
			continue
		}
		res = append(res, PairingInfo{
			ServiceIdentity: shipapi.NewServiceIdentity(ski, "", ""),
			Source:          PairingSourceSki,
		})
	}

	slices.SortFunc(res, func(a, b PairingInfo) int {
		return strings.Compare(a.SKI+a.ShipID, b.SKI+b.ShipID)
	})

	return res
}

// RemovePairing removes a single pairing identified by ship id or ski and revokes its trust
func (c *EEBus) RemovePairing(id string) bool {
	c.mux.Lock()
	idx := slices.IndexFunc(c.paired, func(i shipapi.ServiceIdentity) bool {
		return (i.ShipID != "" && i.ShipID == id) || (i.SKI != "" && i.SKI == id)
	})
	var identity shipapi.ServiceIdentity
	if idx >= 0 {
		identity = c.paired[idx]
		c.paired = slices.Delete(c.paired, idx, idx+1)
		c.persistPairings()
	}
	c.mux.Unlock()

	if idx < 0 {
		return false
	}

	// release mutex before cross-layer call, see UnregisterDevice
	c.service.UnregisterRemoteService(identity)
	return true
}

// persistPairings stores the current pairings (mux must be held)
func (c *EEBus) persistPairings() {
	if err := storeTrustedDevices(c.paired); err != nil {
		c.log.ERROR.Printf("persisting pairings: %v", err)
	}
}

// pairedIndex returns the index of the pairing matching identity, or -1 (mux must be held)
func (c *EEBus) pairedIndex(identity shipapi.ServiceIdentity) int {
	return slices.IndexFunc(c.paired, func(i shipapi.ServiceIdentity) bool {
		return (identity.Fingerprint != "" && identity.Fingerprint == i.Fingerprint) ||
			(identity.ShipID != "" && identity.ShipID == i.ShipID) ||
			(identity.SKI != "" && identity.SKI == i.SKI)
	})
}

// upsertPairing adds or updates a pairing and persists it (mux must be held)
func (c *EEBus) upsertPairing(identity shipapi.ServiceIdentity) {
	if idx := c.pairedIndex(identity); idx >= 0 {
		if c.paired[idx] == identity {
			return
		}
		c.paired[idx] = identity
	} else {
		c.paired = append(c.paired, identity)
	}
	c.persistPairings()
}

// clientsFor returns the devices registered for ski, including devices registered
// without ski when ski is a SHIP-paired device (mux must be held)
func (c *EEBus) clientsFor(ski string) []Device {
	res := c.clients[ski]
	if ski != "" && slices.ContainsFunc(c.paired, func(i shipapi.ServiceIdentity) bool { return i.SKI == ski }) {
		res = append(slices.Clone(res), c.clients[""]...)
	}
	return res
}

// RegisterDevice subscribes a device to the remote service with given ski.
// An empty ski subscribes to the device paired via the SHIP Pairing Service.
func (c *EEBus) RegisterDevice(ski, ip string, device Device) error {
	ski = shiputil.NormalizeSKI(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.ski {
		return errors.New("device ski can not be identical to host ski")
	}

	// trust for the paired device is established by pairing, not by configuration
	if ski != "" {
		identity := shipapi.NewServiceIdentity(ski, "", "")
		if len(ip) > 0 {
			identity.IPv4 = ip
		}
		c.service.RegisterRemoteService(identity)
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	c.clients[ski] = append(c.clients[ski], device)

	// the remote service may already be connected
	connected := c.connected[ski]
	if ski == "" {
		connected = slices.ContainsFunc(c.paired, func(i shipapi.ServiceIdentity) bool {
			return i.SKI != "" && c.connected[i.SKI]
		})
	}
	if connected {
		device.Connect(true)
	}

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
			// The paired device (empty ski) stays trusted until unpaired.
			if ski != "" {
				defer c.service.UnregisterRemoteService(shipapi.NewServiceIdentity(ski, "", ""))
			}
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

func (c *EEBus) Shutdown() {
	c.service.Shutdown()
}

// Use case callback
func (c *EEBus) ucCallback(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.DEBUG.Printf("ski %s event %s", ski, event)

	for _, client := range c.clientsFor(ski) {
		client.UseCaseEvent(device, entity, event)
	}
}

// EEBUSServiceHandler

func (c *EEBus) connect(identity shipapi.ServiceIdentity, connected bool) {
	action := map[bool]string{true: "connected", false: "disconnected"}[connected]
	c.log.DEBUG.Printf("ski %s %s", identity.SKI, action)

	c.mux.Lock()
	defer c.mux.Unlock()

	// learn the ski of a device paired via the SHIP Pairing Service
	if c.pairedIndex(identity) >= 0 {
		c.upsertPairing(identity)
	}

	c.connected[identity.SKI] = connected

	for _, client := range c.clientsFor(identity.SKI) {
		client.Connect(connected)
	}
}

func (c *EEBus) RemoteServiceConnected(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity) {
	c.connect(identity, true)
}

func (c *EEBus) RemoteServiceDisconnected(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity) {
	c.connect(identity, false)
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
func (c *EEBus) ServiceUpdated(identity shipapi.ServiceIdentity) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.pairedIndex(identity) >= 0 {
		c.upsertPairing(identity)
	}
}

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

// SHIP Pairing Service events: the trusted device identity is persisted so it
// reconnects across restarts and is routed to consumers registered without ski
func (c *EEBus) ServiceAutoTrusted(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity) {
	c.log.INFO.Printf("service trusted: %s", identity.ShipID)

	c.mux.Lock()
	defer c.mux.Unlock()
	c.upsertPairing(identity)
}

func (c *EEBus) ServiceAutoTrustFailed(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity, reason error) {
	c.log.INFO.Printf("service trust failed: %s: %v", identity.ShipID, reason)
}

func (c *EEBus) ServiceAutoTrustRemoved(service eebusapi.ServiceInterface, identity shipapi.ServiceIdentity, reason string) {
	c.log.INFO.Printf("service trust removed: %s: %s", identity.ShipID, reason)

	c.mux.Lock()
	defer c.mux.Unlock()

	if idx := c.pairedIndex(identity); idx >= 0 {
		c.paired = slices.Delete(c.paired, idx, idx+1)
		c.persistPairings()
	}
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
