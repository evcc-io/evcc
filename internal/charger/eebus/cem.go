// +build eebus

package eebus

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/amp-x/eebus"
	"github.com/amp-x/eebus/app"
	"github.com/amp-x/eebus/cert"
	"github.com/amp-x/eebus/communication"
	"github.com/amp-x/eebus/server"
	"github.com/amp-x/eebus/ship"
	"github.com/amp-x/eebus/spine"
	"github.com/amp-x/eebus/spine/model"
	"github.com/andig/evcc/util"
)

// Instance is the EEBus CEM listener instance
// This is needed since EEBus CEM is managing all the chargers its connected to
var Instance *CEM

// CEM singleton implementing the EEBus CEM
type CEM struct {
	mux     sync.Mutex
	log     *util.Logger
	srv     *eebus.Server
	hems    spine.Device
	id      string
	clients map[string]func(cc *communication.ConnectionController)
}

// New creates the CEM
func New(log *util.Logger, cert tls.Certificate) (*CEM, error) {
	id := server.UniqueID{Prefix: "evcc"}.String()

	details := communication.ManufacturerDetails{
		DeviceName:    "EVCC",
		DeviceCode:    "EVCC_HEMS_01",
		DeviceAddress: "EVCC_HEMS",
		BrandName:     "EVCC",
	}

	srv := &eebus.Server{
		Log:         log.TRACE,
		Addr:        ":4712",
		Path:        "/ship/",
		Certificate: cert,
		ID:          id,
		Brand:       details.BrandName,
		Model:       details.DeviceCode,
		Type:        string(model.DeviceTypeEnumTypeEnergyManagementSystem),
		Register:    true,
	}

	hems := app.HEMS(details)

	c := &CEM{
		log:     log,
		srv:     srv,
		hems:    hems,
		id:      id,
		clients: make(map[string]func(cc *communication.ConnectionController)),
	}

	zc, err := srv.Announce()
	if err != nil {
		return nil, err
	}
	// defer zc.Shutdown()
	_ = zc

	return c, nil
}

func (c *CEM) Register(ski string, onConnect func(cc *communication.ConnectionController)) {
	c.mux.Lock()
	defer c.mux.Unlock()

	ski = strings.ReplaceAll(ski, "-", "")
	c.log.TRACE.Printf("registering ski: %s", ski)

	c.clients[ski] = onConnect
}

func (c *CEM) Run() {
	ln := &server.Listener{
		Log:          c.log.TRACE,
		AccessMethod: c.id,
		Handler:      c.onConnect,
	}

	if err := c.srv.Listen(ln, c.onCertificate); err != nil {
		c.log.ERROR.Println("eebus listen:", err)
	}
}

func (c *CEM) onConnect(conn ship.Conn) error {
	c.mux.Lock()

	var cb func(cc *communication.ConnectionController)
	for _, onConnect := range c.clients {
		cb = onConnect
	}

	c.mux.Unlock()

	// TODO link connection to charger
	if cb != nil {
		ctrl := communication.NewConnectionController(c.log.TRACE, conn, c.hems)
		cb(ctrl)

		c.log.TRACE.Println("booting")
		err := ctrl.Boot()
		c.log.TRACE.Println("booting:", err)

		return err
	}

	return errors.New("client not registered")
}

func (c *CEM) onCertificate(leaf *x509.Certificate) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	ski, err := cert.SkiFromX509(leaf)
	if err != nil {
		return err
	}

	c.log.TRACE.Printf("verifying client ski: %s", ski)

	for allowed := range c.clients {
		if ski == allowed {
			return nil
		}
	}

	return fmt.Errorf("client ski not allowed: %s", ski)
}
