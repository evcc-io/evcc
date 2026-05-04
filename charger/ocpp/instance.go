package ocpp

import (
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
	once        sync.Once
	instance    *CS
	port        = 8887
	externalUrl string
)

// Forwarder hooks installed by the forwarder feature in init().  Hooks are
// optional; when nil the wrapped ws.Server behaves like a plain ws.NewServer().
// Read-only access is goroutine-safe because hooks are assigned exactly once
// at package init() time, before any charger connects.
var (
	chargerConnectHook    func(ws.Channel)
	chargerDisconnectHook func(ws.Channel)
	chargerMessageHook    func(ws.Channel, []byte) bool
)

// interceptingServer wraps ws.Server and routes connect / disconnect / raw-frame
// events through the package-level forwarder hooks.  msgHook returns true to
// bypass the OCPP message handler (used when upstream is the authoritative
// responder).
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

	// Replace protocol: http -> ws, https -> wss
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	u.Host = fmt.Sprintf("%s:%d", strings.Split(u.Host, ":")[0], 8887) // deliberately fixed, port configurability only for testing

	return u.String()
}

// CurrentConfig returns the current runtime OCPP configuration.
func CurrentConfig() Config {
	return Config{Port: port}
}

// Init initializes the OCPP server
func Init(cfg Config, networkExternalUrl string) {
	port = cfg.Port
	externalUrl = networkExternalUrl
}

func Instance() *CS {
	once.Do(func() {
		log := util.NewLogger("ocpp")

		server := &interceptingServer{Server: ws.NewServer()}
		server.SetCheckOriginHandler(func(r *http.Request) bool { return true })

		dispatcher := ocppj.NewDefaultServerDispatcher(ocppj.NewFIFOQueueMap(0))
		dispatcher.SetTimeout(Timeout)

		endpoint := ocppj.NewServer(server, dispatcher, nil, core.Profile, remotetrigger.Profile, smartcharging.Profile, security.Profile, firmware.Profile)
		endpoint.SetInvalidMessageHook(func(client ws.Channel, err *ocpp.Error, rawMessage string, parsedFields []any) *ocpp.Error {
			log.ERROR.Printf("%v (%s)", err, rawMessage)
			return nil
		})

		cs := ocpp16.NewCentralSystem(endpoint, server)

		instance = &CS{
			log:           log,
			regs:          make(map[string]*registration),
			CentralSystem: cs,
			server:        server,
		}

		instance.txnId.Store(time.Now().UTC().Unix())

		ocppj.SetLogger(instance)

		cs.SetCoreHandler(instance)
		cs.SetSecurityHandler(instance)
		cs.SetFirmwareManagementHandler(instance)
		cs.SetNewChargePointHandler(instance.NewChargePoint)
		cs.SetChargePointDisconnectedHandler(instance.ChargePointDisconnected)

		go instance.errorHandler(cs.Errors())
		go cs.Start(port, "/{ws}")

		// wait for server to start
		for range time.Tick(10 * time.Millisecond) {
			if dispatcher.IsRunning() {
				break
			}
		}
	})

	return instance
}
