package ocpp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/security"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

type Config struct {
	Port int `json:"port"`
}

// ForwarderRule maps a station ID (or "*" for all chargers) to an upstream OCPP server URL.
type ForwarderRule struct {
	StationID         string `json:"stationId" yaml:"stationId"`
	UpstreamURL       string `json:"upstreamUrl" yaml:"upstreamUrl"`
	Password          string `json:"password,omitempty" yaml:"password,omitempty"`
	UpstreamStationID string `json:"upstreamStationId,omitempty" yaml:"upstreamStationId,omitempty"`
	Username          string `json:"username,omitempty" yaml:"username,omitempty"`
	Insecure          bool   `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	CaCert            string `json:"caCert,omitempty" yaml:"caCert,omitempty"`
	ReadOnly          bool   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
}

func (r ForwarderRule) Redacted() ForwarderRule {
	r.Password = util.Masked(r.Password)
	r.CaCert = util.Masked(r.CaCert)
	return r
}

var (
	instance    *CS
	started     func() error // memoized listen; set in NewServer, runs once
	port        = 8887
	boundPort   int
	externalUrl string
)

// Forwarder hooks, nil unless the forwarder is built in (set once in init()
// before any charger connects, so reads need no lock).
var (
	chargerConnectHook    func(ws.Channel)
	chargerDisconnectHook func(ws.Channel)
	chargerMessageHook    func(ws.Channel, []byte) bool
)

// interceptingServer routes connect/disconnect/message events through the
// forwarder hooks. The message hook returns true to bypass evcc's OCPP handler.
type interceptingServer struct {
	ws.Server
}

func (s *interceptingServer) SetMessageHandler(handler ws.MessageHandler) {
	s.Server.SetMessageHandler(func(ch ws.Channel, data []byte) error {
		if chargerMessageHook != nil && chargerMessageHook(ch, data) {
			return nil
		}
		return handler(ch, data)
	})
}

func (s *interceptingServer) SetNewClientHandler(handler ws.ConnectedHandler) {
	s.Server.SetNewClientHandler(func(ch ws.Channel) {
		if chargerConnectHook != nil {
			chargerConnectHook(ch)
		}
		handler(ch)
	})
}

func (s *interceptingServer) SetDisconnectedClientHandler(handler func(ws.Channel)) {
	s.Server.SetDisconnectedClientHandler(func(ch ws.Channel) {
		if chargerDisconnectHook != nil {
			chargerDisconnectHook(ch)
		}
		handler(ch)
	})
}

// Port returns the TCP port the central system is bound to. With the default
// configuration this equals the configured port; when port 0 is configured
// (as in tests) it is the OS-assigned ephemeral port. It returns 0 while the
// server is not bound.
func Port() int {
	return boundPort
}

// GetStatus returns the OCPP runtime status
func GetStatus() Status {
	if instance == nil {
		return Status{}
	}
	return instance.status()
}

// ExternalUrl returns the auto-generated OCPP external URL based on network external URL
func ExternalUrl() string {
	if externalUrl == "" {
		return ""
	}

	u, err := url.Parse(externalUrl)
	if err != nil {
		return ""
	}

	u.Scheme = "ws"
	u.Host = fmt.Sprintf("%s:%d", strings.Split(u.Host, ":")[0], 8887) // deliberately fixed, port configurability only for testing

	return u.String()
}

// CurrentConfig returns the current runtime OCPP configuration.
func CurrentConfig() Config {
	return Config{Port: port}
}

// NewServer builds the OCPP central system without starting it.
func NewServer(cfg Config, networkExternalUrl string) {
	port = cfg.Port
	externalUrl = networkExternalUrl

	log := util.NewLogger("ocpp")

	server := &interceptingServer{Server: ws.NewServer()}
	server.SetCheckOriginHandler(func(r *http.Request) bool { return true })

	dispatcher := ocppj.NewDefaultServerDispatcher(ocppj.NewFIFOQueueMap(0))

	endpoint := ocppj.NewServer(server, dispatcher, nil, core.Profile, remotetrigger.Profile, smartcharging.Profile, security.Profile, firmware.Profile)
	endpoint.SetInvalidMessageHook(func(client ws.Channel, err *ocpp.Error, rawMessage string, parsedFields []any) *ocpp.Error {
		log.ERROR.Printf("%v (%s)", err, rawMessage)
		return nil
	})

	cs := ocpp16.NewCentralSystem(endpoint, server)

	inst := &CS{
		log:           log,
		regs:          make(map[string]*registration),
		CentralSystem: cs,
		server:        server,
		dispatcher:    dispatcher,
	}

	inst.txnId.Store(time.Now().UTC().Unix())

	ocppj.SetLogger(inst)

	cs.SetCoreHandler(inst)
	cs.SetSecurityHandler(inst)
	cs.SetFirmwareManagementHandler(inst)
	cs.SetNewChargePointHandler(inst.NewChargePoint)
	cs.SetChargePointDisconnectedHandler(inst.ChargePointDisconnected)

	// wire the start memo before publishing instance, so Instance() never sees
	// a non-nil instance with a nil started
	started = sync.OnceValue(inst.listen)
	instance = inst
}

// listen starts the central system and blocks until it has bound its port.
func (cs *CS) listen() error {
	cs.dispatcher.SetTimeout(Timeout)

	go cs.errorHandler(cs.Errors())
	go cs.CentralSystem.Start(port, "/{ws}")

	// wait for server to bind
	tick := time.Tick(10 * time.Millisecond)
	timeout := time.After(10 * time.Second)
	for cs.server.Addr() == nil {
		select {
		case <-tick:
		case <-timeout:
			return errors.New("timeout waiting for server to bind")
		}
	}

	boundPort = cs.server.Addr().Port
	return nil
}

// Instance returns the central system, starting it once on first call. It
// returns an error if the server fails to bind its port.
func Instance() (*CS, error) {
	if instance == nil {
		return nil, errors.New("ocpp not configured")
	}
	return instance, started()
}
