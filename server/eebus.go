package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/evcc-io/eebus/cert"
	"github.com/evcc-io/eebus/communication"
	"github.com/evcc-io/eebus/mdns"
	"github.com/evcc-io/eebus/server"
	"github.com/evcc-io/eebus/ship"
	"github.com/evcc-io/eebus/spine/model"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
	"github.com/libp2p/zeroconf/v2"
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
	mux                sync.Mutex
	log                *util.Logger
	srv                *server.Server
	id                 string
	zc                 *zeroconf.Server
	clients            map[string]EEBusClientCBs
	ipaddress          map[string]string
	connectedClients   map[string]ship.Conn
	discoveredClients  map[string]*zeroconf.ServiceEntry
	clientInConnection map[string]bool

	browseMDNSRunning bool
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

	log := util.NewLogger("eebus")

	if len(cc.ShipID) == 0 {
		var err error
		protectedID, err := machine.ProtectedID("evcc-eebus")
		if err != nil {
			return nil, err
		}

		cc.ShipID, err = ship.UniqueIDWithProtectedID(details.BrandName, protectedID)
		if err != nil {
			return nil, err
		}
	}

	cert, err := tls.X509KeyPair(cc.Certificate.Public, cc.Certificate.Private)
	if err != nil {
		return nil, err
	}

	srv := &server.Server{
		Log:         log.TRACE,
		Addr:        cc.Uri,
		Path:        "/ship/",
		Certificate: cert,
		ID:          cc.ShipID,
		Interfaces:  cc.Interfaces,
		Brand:       details.BrandName,
		Model:       details.DeviceCode,
		Type:        string(model.DeviceTypeEnumTypeEnergyManagementSystem),
		Register:    true,
	}

	zc, err := srv.Announce()
	if err != nil {
		return nil, err
	}

	c := &EEBus{
		zc:                 zc,
		log:                log,
		srv:                srv,
		id:                 cc.ShipID,
		clients:            make(map[string]EEBusClientCBs),
		ipaddress:          make(map[string]string),
		connectedClients:   make(map[string]ship.Conn),
		discoveredClients:  make(map[string]*zeroconf.ServiceEntry),
		clientInConnection: make(map[string]bool),
	}

	return c, nil
}

func (c *EEBus) DeviceInfo() communication.ManufacturerDetails {
	return EEBUSDetails
}

func (c *EEBus) Register(ski, ip string, shipConnectHandler func(string, ship.Conn) error, shipDisconnectHandler func(string)) {
	ski = strings.ReplaceAll(ski, "-", "")
	ski = strings.ReplaceAll(ski, " ", "")
	ski = strings.ToLower(ski)
	c.log.TRACE.Printf("registering ski: %s", ski)

	c.mux.Lock()
	c.clients[ski] = EEBusClientCBs{onConnect: shipConnectHandler, onDisconnect: shipDisconnectHandler}
	if ip != "" {
		c.ipaddress[ski] = ip
	}
	c.mux.Unlock()

	// maybe the SKI is already discovered
	_ = c.handleDiscoveredSKI(ski)
}

func (c *EEBus) Run() {
	go c.browseMDNS()

	ln := &server.Listener{
		Log:          c.log.TRACE,
		AccessMethod: c.id,
		Handler:      c.shipHandler,
	}

	if err := c.srv.Listen(ln, c.certificateHandler); err != nil {
		c.log.ERROR.Println("eebus listen:", err)
	}
}

func (c *EEBus) browseMDNS() {
	c.browseMDNSRunning = true

	// Let's start from scratch
	c.mux.Lock()
	for k := range c.discoveredClients {
		delete(c.discoveredClients, k)
	}
	c.mux.Unlock()

	entries := make(chan *zeroconf.ServiceEntry)
	defer close(entries)

	go c.discoverDNS(entries, func(entry *zeroconf.ServiceEntry) {
		c.addDisoveredEntry(entry)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := zeroconf.Browse(ctx, ship.ZeroconfType, ship.ZeroconfDomain, entries); err != nil {
		panic(fmt.Errorf("failed to browse: %w", err))
	}
}

func (c *EEBus) discoverDNS(results <-chan *zeroconf.ServiceEntry, connector func(*zeroconf.ServiceEntry)) {
	for entry := range results {
		c.log.TRACE.Println("mDNS:", entry.HostName, entry.AddrIPv4, entry.Text)

		connector(entry)
	}

	// The mDNS Browse has timed out
	c.browseMDNSRunning = false

	c.browseMissingClients()
}

func (c *EEBus) Shutdown() {
	c.zc.Shutdown()
}

func (c *EEBus) addDisoveredEntry(entry *zeroconf.ServiceEntry) {
	// we need to get the SKI only
	svc, err := mdns.NewFromDNSEntry(entry)

	if err == nil {
		if entry.Text == nil {
			c.log.TRACE.Printf("Ignoring discovered mDNS entry as it has no TXT record: %s", entry.HostName)
			return
		}

		c.patchMdnsEntryWithProvidedIP(svc.SKI, entry)

		if entry.AddrIPv4 == nil && entry.AddrIPv6 == nil {
			c.log.TRACE.Printf("Ignoring discovered mDNS entry as it has no IPv4 and IPv6 address: %s", entry.HostName)
			return
		}

		c.mux.Lock()
		c.discoveredClients[svc.SKI] = entry
		c.mux.Unlock()

		// maybe the SKI is already registered
		_ = c.handleDiscoveredSKI(svc.SKI)
	} else {
		c.log.TRACE.Printf("%s: could not create ship service from DNS entry: %v", entry.HostName, err)
	}
}

func (c *EEBus) handleDiscoveredSKI(ski string) error {
	c.mux.Lock()
	_, connected := c.connectedClients[ski]
	_, registered := c.clients[ski]
	entry, discovered := c.discoveredClients[ski]
	_, connecting := c.clientInConnection[ski]
	c.mux.Unlock()

	c.log.TRACE.Printf("client %s connected %t, registered %t, discovered %t, connecting %t", ski, connected, registered, discovered, connecting)

	if !connected && discovered && registered && !connecting {
		c.mux.Lock()
		c.clientInConnection[ski] = true
		c.mux.Unlock()
		if err := c.connectDiscoveredEntry(ski, entry); err != nil {
			c.mux.Lock()
			delete(c.connectedClients, ski)
			delete(c.clientInConnection, ski)
			c.mux.Unlock()
			return err
		}
		c.mux.Lock()
		delete(c.clientInConnection, ski)
		c.mux.Unlock()
	}

	return nil
}

// add the IP address from the charger configuration to the mDNS entry if it is missing
func (c *EEBus) patchMdnsEntryWithProvidedIP(ski string, entry *zeroconf.ServiceEntry) {
	address, exists := c.ipaddress[ski]
	if entry.AddrIPv4 == nil && exists && address != "" {
		if ip := net.ParseIP(address); ip != nil {
			entry.AddrIPv4 = []net.IP{ip}
		}
	}
}

func (c *EEBus) connectDiscoveredEntry(ski string, entry *zeroconf.ServiceEntry) error {
	c.patchMdnsEntryWithProvidedIP(ski, entry)

	svc, err := mdns.NewFromDNSEntry(entry)

	var conn ship.Conn
	if err == nil {
		c.log.TRACE.Printf("%s: client connect", entry.HostName)
		conn, err = svc.Connect(c.log.TRACE, c.id, c.srv.Certificate, c.shipCloseHandler)
	}

	if err != nil {
		c.log.TRACE.Printf("%s: client done: %v", entry.HostName, err)
		return err
	}

	err = c.shipHandler(svc.SKI, conn)
	if err != nil {
		log.FATAL.Fatalf("%s: error calling shipHandler: %v", entry.HostName, err)
		return err
	}

	return nil
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
			c.log.TRACE.Printf("client ski found")
			return nil
		}
	}

	c.log.TRACE.Printf("client ski not found!")

	return fmt.Errorf("client ski not allowed: %s", ski)
}

func (c *EEBus) shipHandler(ski string, conn ship.Conn) error {
	for client, cb := range c.clients {
		if client == ski {
			c.mux.Lock()
			currentConnection, found := c.connectedClients[ski]
			c.mux.Unlock()
			connect := true
			c.log.TRACE.Printf("client %s found? %t", ski, found)
			if found {
				if currentConnection.IsConnectionClosed() {
					c.log.TRACE.Printf("client has closed connection")
				} else {
					c.log.TRACE.Printf("client has no closed connection")
					connect = false
				}
			}
			c.log.TRACE.Printf("client %s connect? %t", ski, connect)
			if connect {
				c.mux.Lock()
				c.connectedClients[ski] = conn
				c.mux.Unlock()
				return cb.onConnect(ski, conn)
			}
		}
	}

	return errors.New("client not registered")
}

// handles connection closed
func (c *EEBus) shipCloseHandler(ski string) {
	c.mux.Lock()
	if conn, ok := c.connectedClients[ski]; ok {
		for clientSki, client := range c.clients {
			if clientSki != ski {
				continue
			}

			if conn.IsConnectionClosed() {
				c.log.TRACE.Printf("close client %s connection", ski)
				client.onDisconnect(ski)
			}

			// always remove client on close
			delete(c.connectedClients, ski)

			break
		}
	}

	c.mux.Unlock()
	c.browseMissingClients()
}

// search for registered but not connected clients right away
// this fixes registered clients closing the connection whysoever
func (c *EEBus) browseMissingClients() {
	if c.browseMDNSRunning {
		return
	}

	if len(c.clients) == len(c.connectedClients) {
		return
	}

	c.browseMDNSRunning = true

	// first try the discovered client, if that not works, browse again
	foundMissingClients := len(c.clients) - len(c.connectedClients)
	var failedClientSKIs []string
	for ski, entry := range c.discoveredClients {
		if _, ok := c.clients[ski]; !ok {
			continue
		}

		if _, ok := c.connectedClients[ski]; !ok {
			c.log.TRACE.Printf("%s: client not connected, trying to connect\n", entry.HostName)
			if err := c.handleDiscoveredSKI(ski); err != nil {
				failedClientSKIs = append(failedClientSKIs, ski)
				continue
			}
			foundMissingClients--
		}
	}

	c.mux.Lock()
	for _, ski := range failedClientSKIs {
		delete(c.discoveredClients, ski)
	}
	c.mux.Unlock()

	c.browseMDNSRunning = false

	if foundMissingClients > 0 {
		c.browseMDNS()
	}
}
