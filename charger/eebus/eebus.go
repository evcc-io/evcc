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
	"strings"
	"sync"

	"github.com/enbility/cemd/cem"
	"github.com/enbility/cemd/emobility"
	"github.com/enbility/eebus-go/service"
	"github.com/enbility/eebus-go/spine/model"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
)

const (
	EEBUSBrandName  string = "EVCC"
	EEBUSModel      string = "HEMS"
	EEBUSDeviceCode string = "EVCC_HEMS_01" // used as common name in cert generation
)

type EEBusClientCBs struct {
	onConnect    func(string) // , ship.Conn) error
	onDisconnect func(string)
}

type EEBus struct {
	Cem *cem.CemImpl

	mux sync.Mutex
	log *util.Logger

	SKI string

	clients map[string]EEBusClientCBs
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
	configuration, err := service.NewConfiguration(
		EEBUSBrandName, EEBUSBrandName, EEBUSModel, serial,
		model.DeviceTypeTypeEnergyManagementSystem, port, certificate, 230,
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
		clients: make(map[string]EEBusClientCBs),
		SKI:     ski,
	}

	c.Cem = cem.NewCEM(configuration, c, c)
	if err := c.Cem.Setup(); err != nil {
		return nil, err
	}
	c.Cem.EnableEmobility(emobility.EmobilityConfiguration{
		CoordinatedChargingEnabled: false,
	})

	return c, nil
}

func (c *EEBus) RegisterEVSE(ski, ip string, connectHandler func(string), disconnectHandler func(string), dataProvider emobility.EmobilityDataProvider) *emobility.EMobilityImpl {
	ski = strings.ReplaceAll(ski, "-", "")
	ski = strings.ReplaceAll(ski, " ", "")
	ski = strings.ToLower(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	if ski == c.SKI {
		c.log.FATAL.Fatal("The charger SKI can not be identical to the SKI of evcc!")
	}

	serviceDetails := service.NewServiceDetails(ski)
	serviceDetails.SetIPv4(ip)

	c.mux.Lock()
	defer c.mux.Unlock()
	c.clients[ski] = EEBusClientCBs{onConnect: connectHandler, onDisconnect: disconnectHandler}

	return c.Cem.RegisterEmobilityRemoteDevice(serviceDetails, dataProvider)
}

func (c *EEBus) Run() {
	c.Cem.Start()
}

func (c *EEBus) Shutdown() {
	c.Cem.Shutdown()
}

// EEBUSServiceHandler

// report the Ship ID of a newly trusted connection
func (c *EEBus) RemoteServiceShipIDReported(service *service.EEBUSService, ski string, shipID string) {
	// we should associated the Ship ID with the SKI and store it
	// so the next connection can start trusted
	c.log.DEBUG.Println("SKI", ski, "has Ship ID:", shipID)
}

func (c *EEBus) RemoteSKIConnected(service *service.EEBUSService, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	client, exists := c.clients[ski]
	if !exists {
		return
	}
	client.onConnect(ski)
}

func (c *EEBus) RemoteSKIDisconnected(service *service.EEBUSService, ski string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	client, exists := c.clients[ski]
	if !exists {
		return
	}
	client.onDisconnect(ski)
}

func (h *EEBus) ReportServiceShipID(ski string, shipdID string) {}

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
	return service.CreateCertificate("", EEBUSBrandName, "DE", EEBUSDeviceCode)
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
