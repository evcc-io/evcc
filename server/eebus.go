package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/eebus/cert"
	"github.com/evcc-io/eebus/communication"
	"github.com/evcc-io/eebus/mdns"
	"github.com/evcc-io/eebus/server"
	"github.com/evcc-io/eebus/ship"
	"github.com/evcc-io/eebus/spine/model"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"
	"github.com/grandcat/zeroconf"
)

var EEBUSDetails = communication.ManufacturerDetails{
	BrandName:     "EVCC",
	DeviceName:    "EVCC",
	DeviceCode:    "EVCC_HEMS_01",
	DeviceAddress: "EVCC_HEMS",
}

type EEBusClientCBs struct {
	onConnect    func(string, ship.Conn) error
	onDisconnect func(string)
}

type EEBus struct {
	mux               sync.Mutex
	log               logx.Logger
	srv               *server.Server
	id                string
	clients           map[string]EEBusClientCBs
	connectedClients  map[string]ship.Conn
	discoveredClients map[string]*zeroconf.ServiceEntry
}

var EEBusInstance *EEBus

func NewEEBus(other map[string]interface{}) (*EEBus, error) {
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

	// if !sponsor.IsAuthorized() {
	// 	return nil, errors.New("eebus requires evcc sponsorship, register at https://cloud.evcc.io")
	// }

	details := EEBusInstance.DeviceInfo()

	log := logx.NewModule("eebus")
	id := server.UniqueID{Prefix: details.BrandName}.String()
	if len(cc.ShipID) > 0 {
		id = cc.ShipID
	}

	cert, err := tls.X509KeyPair(cc.Certificate.Public, cc.Certificate.Private)
	if err != nil {
		return nil, err
	}

	srv := &server.Server{
		Log:         logx.NewPrintAdapter(logx.TraceLevel(log)), // match traceLogAdapter()
		Addr:        cc.Uri,
		Path:        "/ship/",
		Certificate: cert,
		ID:          id,
		Interfaces:  cc.Interfaces,
		Brand:       details.BrandName,
		Model:       details.DeviceCode,
		Type:        string(model.DeviceTypeEnumTypeEnergyManagementSystem),
		Register:    true,
	}

	if _, err = srv.Announce(); err != nil {
		return nil, err
	}

	c := &EEBus{
		log:               log,
		srv:               srv,
		id:                id,
		clients:           make(map[string]EEBusClientCBs),
		connectedClients:  make(map[string]ship.Conn),
		discoveredClients: make(map[string]*zeroconf.ServiceEntry),
	}

	return c, nil
}

func (c *EEBus) traceLogAdapter() logx.Printer {
	return logx.NewPrintAdapter(logx.TraceLevel(c.log))
}

func (c *EEBus) DeviceInfo() communication.ManufacturerDetails {
	return EEBUSDetails
}

func (c *EEBus) Register(ski string, shipConnectHandler func(string, ship.Conn) error, shipDisconnectHandler func(string)) {
	ski = strings.ReplaceAll(ski, "-", "")
	logx.Trace(c.log, "msg", "registering ski: %s", ski)

	c.mux.Lock()
	c.clients[ski] = EEBusClientCBs{onConnect: shipConnectHandler, onDisconnect: shipDisconnectHandler}
	c.mux.Unlock()

	// maybe the SKI is already discovered
	c.handleDiscoveredSKI(ski)
}

func (c *EEBus) Run() {
	entries := make(chan *zeroconf.ServiceEntry)
	go c.discoverDNS(entries, func(entry *zeroconf.ServiceEntry) {
		c.addDisoveredEntry(entry)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		panic(fmt.Errorf("mDNS: failed initializing resolver: %w", err))
	}

	if err = resolver.Browse(ctx, ship.ZeroconfType, ship.ZeroconfDomain, entries); err != nil {
		panic(fmt.Errorf("failed to browse: %w", err))
	}

	ln := &server.Listener{
		Log:          c.traceLogAdapter(),
		AccessMethod: c.id,
		Handler:      c.shipHandler,
	}

	if err := c.srv.Listen(ln, c.certificateHandler); err != nil {
		logx.Error(c.log, "msg", "eebus listen", "error", err)
	}
}

func (c *EEBus) addDisoveredEntry(entry *zeroconf.ServiceEntry) {
	// we need to get the SKI only
	svc, err := mdns.NewFromDNSEntry(entry)

	if err == nil {
		c.mux.Lock()
		c.discoveredClients[svc.SKI] = entry
		c.mux.Unlock()

		// maybe the SKI is already registered
		c.handleDiscoveredSKI(svc.SKI)
	} else {
		logx.Trace(c.log, "msg", "could not create ship service", "from", entry.HostName, "error", err)
	}
}

func (c *EEBus) handleDiscoveredSKI(ski string) {
	c.mux.Lock()

	_, connected := c.connectedClients[ski]
	_, registered := c.clients[ski]
	entry, discovered := c.discoveredClients[ski]

	logx.Trace(c.log, "msg", "client %s connected %t, registered %t, discovered %t ", ski, connected, registered, discovered)

	if !connected && discovered && registered {
		c.mux.Unlock()
		c.connectDiscoveredEntry(entry)
		return
	}

	c.mux.Unlock()
}

func (c *EEBus) connectDiscoveredEntry(entry *zeroconf.ServiceEntry) {
	svc, err := mdns.NewFromDNSEntry(entry)

	var conn ship.Conn
	if err == nil {
		logx.Trace(c.log, "msg", "%s: client connect", entry.HostName)
		conn, err = svc.Connect(c.traceLogAdapter(), c.id, c.srv.Certificate, c.shipCloseHandler)
	}

	if err != nil {
		logx.Trace(c.log, "msg", "%s: client done: %v", entry.HostName, err)
		return
	}

	err = c.shipHandler(svc.SKI, conn)
	if err != nil {
		logx.Error(log, "msg", "shipHandler failed", "error", err, "host", entry.HostName)
		return
	}
}

func (c *EEBus) discoverDNS(results <-chan *zeroconf.ServiceEntry, connector func(*zeroconf.ServiceEntry)) {
	for entry := range results {
		logx.Trace(c.log, "msg", "mDNS discovery", "host", entry.HostName, "ipv4", entry.AddrIPv4, "text", entry.Text)

		for _, typ := range entry.Text {
			if strings.HasPrefix(typ, "type=") && typ == "type=EVSE" {
				connector(entry)
			}
		}
	}
}

func (c *EEBus) certificateHandler(leaf *x509.Certificate) error {
	ski, err := cert.SkiFromX509(leaf)
	if err != nil {
		return err
	}

	logx.Trace(c.log, "msg", "verifying client ski: %s", ski)

	c.mux.Lock()
	defer c.mux.Unlock()

	for client := range c.clients {
		if client == ski {
			logx.Trace(c.log, "msg", "client ski found")
			return nil
		}
	}

	logx.Trace(c.log, "msg", "client ski not found!")

	return fmt.Errorf("client ski not allowed: %s", ski)
}

func (c *EEBus) shipHandler(ski string, conn ship.Conn) error {
	c.mux.Lock()

	for client, cb := range c.clients {
		if client == ski {
			currentConnection, found := c.connectedClients[ski]
			connect := true
			logx.Trace(c.log, "msg", "client %s found? %t", ski, found)
			if found {
				if currentConnection.IsConnectionClosed() {
					logx.Trace(c.log, "msg", "client has closed connection")
					delete(c.connectedClients, ski)
				} else {
					logx.Trace(c.log, "msg", "client has no closed connection")
					connect = false
				}
			}
			logx.Trace(c.log, "msg", "client %s connect? %t", ski, connect)
			if connect {
				c.connectedClients[ski] = conn
				c.mux.Unlock()
				err := cb.onConnect(ski, conn)
				if err != nil {
					c.mux.Lock()
					delete(c.connectedClients, ski)
					c.mux.Unlock()
				}
				return err
			}
		}
	}

	c.mux.Unlock()

	return errors.New("client not registered")
}

// handles connection closed
func (c *EEBus) shipCloseHandler(ski string) {
	c.mux.Lock()

	currentConnection, found := c.connectedClients[ski]
	if found {
		var closeConnection bool
		var clientCB EEBusClientCBs
		for clientSki, client := range c.clients {
			if clientSki == ski && !currentConnection.IsConnectionClosed() {
				clientCB = client
				closeConnection = true
				break
			}
		}
		if closeConnection {
			logx.Trace(c.log, "msg", "close client %s connection", ski)
			clientCB.onDisconnect(ski)
			delete(c.connectedClients, ski)
		}
	}

	c.mux.Unlock()
}
