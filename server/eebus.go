// +build eebus

package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/amp-x/eebus"
	"github.com/amp-x/eebus/cert"
	"github.com/amp-x/eebus/communication"
	"github.com/amp-x/eebus/server"
	"github.com/amp-x/eebus/ship"
	"github.com/amp-x/eebus/spine/model"
	"github.com/andig/evcc/util"
)

type EEBus struct {
	mux     sync.Mutex
	log     *util.Logger
	srv     *eebus.Server
	id      string
	clients map[string]func(string, ship.Conn) error
}

var EEBusInstance *EEBus

func NewEEBus(other map[string]interface{}) (*EEBus, error) {
	cc := struct {
		Uri         string
		Certificate struct {
			Public, Private []byte
		}
	}{
		Uri: ":4712",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// if !sponsor.IsAuthorized() {
	// 	return nil, errors.New("eebus requires evcc sponsorship, register at https://cloud.evcc.io")
	// }

	details := EEBusInstance.DeviceInfo()

	log := util.NewLogger("eebus")
	id := server.UniqueID{Prefix: details.BrandName}.String()

	cert, err := tls.X509KeyPair(cc.Certificate.Public, cc.Certificate.Private)
	if err != nil {
		return nil, err
	}

	srv := &eebus.Server{
		Log:         log.TRACE,
		Addr:        cc.Uri,
		Path:        "/ship/",
		Certificate: cert,
		ID:          id,
		Brand:       details.BrandName,
		Model:       details.DeviceCode,
		Type:        string(model.DeviceTypeEnumTypeEnergyManagementSystem),
		Register:    true,
	}

	if _, err = srv.Announce(); err != nil {
		return nil, err
	}

	c := &EEBus{
		log:     log,
		srv:     srv,
		id:      id,
		clients: make(map[string]func(string, ship.Conn) error),
	}

	return c, nil
}

func (c *EEBus) DeviceInfo() communication.ManufacturerDetails {
	return communication.ManufacturerDetails{
		BrandName:     "EVCC",
		DeviceName:    "EVCC",
		DeviceCode:    "EVCC_HEMS_01",
		DeviceAddress: "EVCC_HEMS",
	}
}

func (c *EEBus) Register(ski string, shipHandler func(string, ship.Conn) error) {
	ski = strings.ReplaceAll(ski, "-", "")
	c.log.TRACE.Printf("registering ski: %s", ski)

	c.mux.Lock()
	c.clients[ski] = shipHandler
	c.mux.Unlock()
}

func (c *EEBus) Run() {
	ln := &server.Listener{
		Log:          c.log.TRACE,
		AccessMethod: c.id,
		Handler:      c.shipHandler,
	}

	if err := c.srv.Listen(ln, c.certificateHandler); err != nil {
		c.log.ERROR.Println("eebus listen:", err)
	}
}

func (c *EEBus) certificateHandler(leaf *x509.Certificate) error {
	ski, err := cert.SkiFromX509(leaf)
	if err != nil {
		return err
	}

	c.log.TRACE.Printf("verifying client ski: %s", ski)

	c.mux.Lock()
	defer c.mux.Unlock()

	for client := range c.clients {
		if client == ski {
			return nil
		}
	}

	return fmt.Errorf("client ski not allowed: %s", ski)
}

func (c *EEBus) shipHandler(ski string, conn ship.Conn) error {
	c.mux.Lock()

	for client, cb := range c.clients {
		if client == ski {
			c.mux.Unlock()
			return cb(ski, conn)
		}
	}

	c.mux.Unlock()

	return errors.New("client not registered")
}
