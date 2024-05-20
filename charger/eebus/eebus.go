package eebus

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"dario.cat/mergo"
	cemdapi "github.com/enbility/cemd/api"
	"github.com/enbility/cemd/cem"
	"github.com/enbility/cemd/ucevcc"
	"github.com/enbility/cemd/ucevcem"
	"github.com/enbility/cemd/ucevsecc"
	"github.com/enbility/cemd/ucevsoc"
	"github.com/enbility/cemd/ucopev"
	"github.com/enbility/cemd/ucoscev"
	eebusapi "github.com/enbility/eebus-go/api"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/cert"
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
	DeviceConnect(device spineapi.DeviceRemoteInterface, event cemdapi.EventType)
	UseCaseEventCB(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event cemdapi.EventType)
}

// EVSE UseCases
type UseCasesEVSE struct {
	EvseCC ucevsecc.UCEVSECCInterface
	EvCC   ucevcc.UCEVCCInterface
	EvCem  ucevcem.UCEVCEMInterface
	EVSoc  ucevsoc.UCEVSOCInterface
	OpEV   ucopev.UCOPEVInterface
	OscEV  ucoscev.UCOSCEVInterface
}

type EEBus struct {
	Cem *cem.Cem

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
		port, certificate, 230, time.Second*4,
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

	c.Cem = cem.NewCEM(configuration, c, c.deviceEventCB, c)
	if err := c.Cem.Setup(); err != nil {
		return nil, err
	}

	evsecc := ucevsecc.NewUCEVSECC(c.Cem.Service, c.evseUsecaseCB)
	c.Cem.AddUseCase(evsecc)

	evcc := ucevcc.NewUCEVCC(c.Cem.Service, c.evseUsecaseCB)
	c.Cem.AddUseCase(evcc)

	evcem := ucevcem.NewUCEVCEM(c.Cem.Service, c.evseUsecaseCB)
	c.Cem.AddUseCase(evcem)

	opev := ucopev.NewUCOPEV(c.Cem.Service, c.evseUsecaseCB)
	c.Cem.AddUseCase(opev)

	oscev := ucoscev.NewUCOSCEV(c.Cem.Service, c.evseUsecaseCB)
	c.Cem.AddUseCase(oscev)

	evsoc := ucevsoc.NewUCEVSOC(c.Cem.Service, c.evseUsecaseCB)
	c.Cem.AddUseCase(evsoc)

	c.evseUC = &UseCasesEVSE{
		EvseCC: evsecc,
		EvCC:   evcc,
		EvCem:  evcem,
		OpEV:   opev,
		OscEV:  oscev,
		EVSoc:  evsoc,
	}

	return c, nil
}

func (c *EEBus) RegisterEVSE(ski string, device EEBUSDeviceInterface) *UseCasesEVSE {
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.SKI {
		c.log.FATAL.Fatal("The charger SKI can not be identical to the SKI of evcc!")
	}

	c.Cem.Service.RegisterRemoteSKI(ski)

	c.mux.Lock()
	defer c.mux.Unlock()
	c.clients[ski] = device

	return c.evseUC
}

func (c *EEBus) Run() {
	c.Cem.Start()
}

func (c *EEBus) Shutdown() {
	c.Cem.Shutdown()
}

// CEMd Callbacks
func (c *EEBus) deviceEventCB(ski string, device spineapi.DeviceRemoteInterface, event cemdapi.EventType) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if client, ok := c.clients[ski]; ok {
		client.DeviceConnect(device, event)
	}
}

// EVSE/EV UseCase CB
func (c *EEBus) evseUsecaseCB(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event cemdapi.EventType) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if client, ok := c.clients[ski]; ok {
		client.UseCaseEventCB(device, entity, event)
	}
}

// EEBUSServiceHandler

// no implementation needed, handled in CEM events
func (c *EEBus) RemoteSKIConnected(service eebusapi.ServiceInterface, ski string) {}

// no implementation needed, handled in CEM events
func (c *EEBus) RemoteSKIDisconnected(service eebusapi.ServiceInterface, ski string) {}

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
		c.Cem.Service.CancelPairingWithSKI(ski)
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

// Certificate helpers

// CreateCertificate returns a newly created EEBUS compatible certificate
func CreateCertificate() (tls.Certificate, error) {
	return cert.CreateCertificate("", EEBUSBrandName, "DE", EEBUSDeviceCode)
}

// pemBlockForKey marshals private key into pem block
func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal ECDSA private key: %w", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, errors.New("unknown private key type")
	}
}

// GetX509KeyPair saves returns the cert and key string values
func GetX509KeyPair(cert tls.Certificate) (string, string, error) {
	var certValue, keyValue string

	out := new(bytes.Buffer)
	err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	if err == nil {
		certValue = out.String()
	}

	if len(certValue) > 0 {
		var pb *pem.Block
		if pb, err = pemBlockForKey(cert.PrivateKey); err == nil {
			out.Reset()
			err = pem.Encode(out, pb)
		}
	}

	if err == nil {
		keyValue = out.String()
	}

	return certValue, keyValue, err
}

// SkiFromX509 extracts SKI from certificate
func skiFromX509(leaf *x509.Certificate) (string, error) {
	if len(leaf.SubjectKeyId) == 0 {
		return "", errors.New("missing SubjectKeyId")
	}
	return fmt.Sprintf("%0x", leaf.SubjectKeyId), nil
}

// SkiFromCert extracts SKI from certificate
func SkiFromCert(cert tls.Certificate) (string, error) {
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return "", errors.New("failed parsing certificate: " + err.Error())
	}
	return skiFromX509(leaf)
}
