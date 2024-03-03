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

	cemdapi "github.com/enbility/cemd/api"
	"github.com/enbility/cemd/cem"
	"github.com/enbility/cemd/ucevcc"
	"github.com/enbility/cemd/ucevcem"
	"github.com/enbility/cemd/ucevsecc"
	"github.com/enbility/cemd/ucevsoc"
	"github.com/enbility/cemd/ucopev"
	"github.com/enbility/eebus-go/api"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/cert"
	"github.com/enbility/ship-go/logging"
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

type Usecases struct {
	Evsecc ucevsecc.UCEVSECCInterface // EVSE Commissioning and Configuration
	EvCC   ucevcc.UCEVCCInterface     // EV Commissioning and Configuration
	EvCem  ucevcem.UCEVCEMInterface   // EV Charging Electricity Measurement
	OpEV   ucopev.UCOPEVInterface     // EV Overload Protection
	EvSoc  ucevsoc.UCEVSOCInterface   // EV State Of Charge
}

type Callbacks struct {
	onConnect    func(spineapi.EntityRemoteInterface)
	onDisconnect func(spineapi.EntityRemoteInterface)
}

type EEBus struct {
	mux sync.Mutex
	log *util.Logger

	SKI     string
	clients map[string]*Callbacks
	cem     *cem.Cem
	uc      *Usecases
}

var Instance *EEBus

func NewServer(other map[string]interface{}) (*EEBus, error) {
	cc := struct {
		Uri         string
		ShipID      string
		Interfaces  []string
		Certificate struct {
			Public, Private []byte
		}
	}{
		Uri: ":4712",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("eebus")

	var err error
	protectedID, err := machine.ProtectedID("evcc-eebus")
	if err != nil {
		return nil, err
	}
	serial := fmt.Sprintf("%s-%0x", "EVCC", protectedID[:8])

	if len(cc.ShipID) != 0 {
		serial = cc.ShipID
	}

	certificate, err := tls.X509KeyPair(cc.Certificate.Public, cc.Certificate.Private)
	if err != nil {
		return nil, err
	}

	_, portValue, err := net.SplitHostPort(cc.Uri)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return nil, err
	}

	// TODO: get the voltage from the site
	configuration, err := api.NewConfiguration(
		EEBUSBrandName, EEBUSBrandName, EEBUSModel, serial,
		model.DeviceTypeTypeEnergyManagementSystem, []model.EntityTypeType{model.EntityTypeTypeCEM},
		port, certificate,
		230, 5*time.Second,
	)
	if err != nil {
		return nil, err
	}

	// for backward compatibility
	configuration.SetAlternateMdnsServiceName("EVCC_HEMS_01")
	configuration.SetAlternateIdentifier(serial)
	configuration.SetInterfaces(cc.Interfaces)
	configuration.SetRegisterAutoAccept(true)

	ski, err := SkiFromCert(certificate)
	if err != nil {
		return nil, err
	}

	c := &EEBus{
		log:     log,
		clients: make(map[string]*Callbacks),
		SKI:     ski,
	}

	c.cem = cem.NewCEM(configuration, c, c.eventHandler, c)

	// create use cases
	service := c.cem.Service
	c.uc = &Usecases{
		Evsecc: ucevsecc.NewUCEVSECC(service, c.eventHandler),
		EvCC:   ucevcc.NewUCEVCC(service, c.eventHandler),
		EvCem:  ucevcem.NewUCEVCEM(service, c.eventHandler),
		OpEV:   ucopev.NewUCOPEV(service, c.eventHandler),
		EvSoc:  ucevsoc.NewUCEVSOC(service, c.eventHandler),
	}

	// register use cases
	c.cem.AddUseCase(c.uc.Evsecc)
	c.cem.AddUseCase(c.uc.EvCC)
	c.cem.AddUseCase(c.uc.EvCem)
	c.cem.AddUseCase(c.uc.OpEV)
	c.cem.AddUseCase(c.uc.EvSoc)

	if err := c.cem.Setup(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *EEBus) eventHandler(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event cemdapi.EventType) {
	c.log.TRACE.Printf("eventHandler: CEM %s %s %v", ski, event, entity)

	c.mux.Lock()
	defer c.mux.Unlock()

	callbacks, ok := c.clients[ski]
	if !ok {
		c.log.TRACE.Printf("eventHandler: CEM ski %s not registered", ski)
		return
	}

	switch event {
	case ucevcc.EvConnected:
		callbacks.onConnect(entity)
	case ucevcc.EvDisconnected:
		callbacks.onDisconnect(entity)
	}
}

func (c *EEBus) RegisterEVSE(ski, ip string, connectHandler, disconnectHandler func(spineapi.EntityRemoteInterface)) (*Usecases, error) {
	ski = NormalizeSki(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.SKI {
		c.log.FATAL.Fatal("charger SKI can not be identical to evcc SKI")
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	if _, ok := c.clients[ski]; ok {
		return nil, errors.New("ski already registered")
	}

	c.clients[ski] = &Callbacks{onConnect: connectHandler, onDisconnect: disconnectHandler}

	return c.uc, nil
}

func (c *EEBus) Run() {
	c.cem.Start()
}

func (c *EEBus) Shutdown() {
	c.cem.Shutdown()
}

// EEBUSServiceHandler

var _ api.ServiceReaderInterface = (*EEBus)(nil)

func (c *EEBus) RemoteSKIConnected(service api.ServiceInterface, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	client, exists := c.clients[ski]
	if !exists {
		return
	}
	_ = client
	// client.onConnect(ski)
}

func (c *EEBus) RemoteSKIDisconnected(service api.ServiceInterface, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	client, exists := c.clients[ski]
	if !exists {
		return
	}
	_ = client
	// client.onDisconnect(ski)
}

func (c *EEBus) VisibleRemoteServicesUpdated(service api.ServiceInterface, entries []shipapi.RemoteService) {
}

func (c *EEBus) ServiceShipIDUpdate(ski string, shipdID string) {
}

func (c *EEBus) ServicePairingDetailUpdate(ski string, detail *shipapi.ConnectionStateDetail) {
}

func (c *EEBus) AllowWaitingForTrust(ski string) bool {
	return true
}

// EEBUS Logging interface

var _ logging.LoggingInterface = (*EEBus)(nil)

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
