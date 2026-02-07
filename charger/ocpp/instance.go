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
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/security"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

type Config struct {
	Port int `json:"port"`
}

var (
	once        sync.Once
	instance    *CS
	port        = 8887
	externalUrl string
)

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

// Init initializes the OCPP server
func Init(cfg Config, networkExternalUrl string) {
	port = cfg.Port
	externalUrl = networkExternalUrl
}

func Instance() *CS {
	once.Do(func() {
		log := util.NewLogger("ocpp")

		server := ws.NewServer()
		server.SetCheckOriginHandler(func(r *http.Request) bool { return true })

		dispatcher := ocppj.NewDefaultServerDispatcher(ocppj.NewFIFOQueueMap(0))
		dispatcher.SetTimeout(Timeout)

		endpoint := ocppj.NewServer(server, dispatcher, nil, core.Profile, remotetrigger.Profile, smartcharging.Profile, security.Profile)
		endpoint.SetInvalidMessageHook(func(client ws.Channel, err *ocpp.Error, rawMessage string, parsedFields []any) *ocpp.Error {
			log.ERROR.Printf("%v (%s)", err, rawMessage)
			return nil
		})

		cs := ocpp16.NewCentralSystem(endpoint, server)

		instance = &CS{
			log:           log,
			regs:          make(map[string]*registration),
			CentralSystem: cs,
		}

		instance.txnId.Store(time.Now().UTC().Unix())

		ocppj.SetLogger(instance)

		cs.SetCoreHandler(instance)
		cs.SetSecurityHandler(instance)
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
