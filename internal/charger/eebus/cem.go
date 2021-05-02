// +build eebus

package eebus

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"sync"

	"github.com/amp-x/eebus"
	"github.com/amp-x/eebus/communication"
	"github.com/amp-x/eebus/server"
	"github.com/amp-x/eebus/ship"
	"github.com/andig/evcc/util"
)

// Instance is the EEBus CEM listener instance
// This is needed since EEBus CEM is managing all the chargers its connected to
var Instance *CEM

// CEM singleton implementing the EEBus CEM
type CEM struct {
	mux        sync.Mutex
	log        *util.Logger
	srv        *eebus.Server
	clientSKIs []string
	sc         *communication.ServiceController
}

// New creates the CEM
func New(log *util.Logger, cert tls.Certificate) (*CEM, error) {
	srv := &eebus.Server{
		Addr:        ":4712",
		Path:        "/ship/",
		Certificate: cert,
		// ID, Brand, Model, Type string
		Register: true,
	}

	c := &CEM{
		log: log,
		srv: srv,
	}

	zc, err := srv.Announce()
	if err != nil {
		return nil, err
	}
	// defer zc.Shutdown()
	_ = zc

	// go func() {
	// 	if err := l.sc.Boot(); err != nil {
	// 		log.FATAL.Fatal(err)
	// 		os.Exit(1)
	// 	}
	// }()

	return c, nil
}

func (c *CEM) AllowSki(ski string) {
	c.mux.Lock()
	c.clientSKIs = append(c.clientSKIs, ski)
	c.mux.Unlock()
}

func (c *CEM) Run() {
	device := communication.ManufacturerDetailsType{
		DeviceName:    "EVCC",
		DeviceCode:    "EVCC_HEMS_01",
		DeviceAddress: "EVCC_HEMS",
		BrandName:     "EVCC",
	}

	c.sc = communication.NewServiceController(c.log.TRACE, device, c.srv.Certificate)

	ln := &server.Listener{
		Log: c.log.TRACE,
		// AccessMethod: c.mdnsID.String(),
		CertificateVerifier: c.onCertificate,
		Handler:             c.onConnect,
	}

	c.srv.Listen(ln)
}

func (c *CEM) onConnect(conn ship.Conn) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.clientSKI == "" {
		return errors.New("missing client ski")
	}

	return c.sc.OnConnect(conn, c.clientSKI)
}

func (c *CEM) onCertificate(certs []*x509.Certificate) error {
	if len(certs) == 0 {
		return errors.New("missing client certificate")
	}

	leaf := certs[0]
	ski := fmt.Sprintf("%0x", leaf.SubjectKeyId)

	c.mux.Lock()
	defer c.mux.Unlock()

	for _, allowed := range c.clientSKIs {
		if ski == allowed {
			c.clientSKI = ski
			return nil
		}
	}

	c.clientSKI = ""

	return fmt.Errorf("client ski not allowed: %s", ski)
}

// EEBUS ServiceController send new loadControl currents
func (c *CEM) SetCurrents(ski string, currentL1, currentL2, currentL3 float64) error {
	return c.sc.SetCurrents(ski, []float64{currentL1, currentL2, currentL3})
}

// EEBUS ServiceController get current dataset
func (c *CEM) GetData(ski string) (*communication.EVSEClientDataType, error) {
	return c.sc.GetEVSEClientData(ski)
}
