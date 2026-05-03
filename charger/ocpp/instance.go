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
	ocpp2 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/authorization"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/data"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/diagnostics"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/display"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/iso15118"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/meter"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/remotecontrol"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/reservation"
	security20 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1/security"
	smartcharging20 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/tariffcost"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

type Config struct {
	Port   int `json:"port"`   // OCPP 1.6 port (default 8887)
	Port20 int `json:"port20"` // OCPP 2.0.1 port (default 8886)
}

var (
	once        sync.Once
	instance    *CS
	port        = 8887
	externalUrl string

	once20     sync.Once
	instance20 *CSMS20
	port20     = 8886
)

// GetStatus returns the OCPP runtime status for both 1.6 and 2.0.1
func GetStatus() Status {
	status := Status{}

	if instance != nil {
		status = instance.status()
	}

	// Append 2.0.1 stations
	if instance20 != nil {
		status.Stations = append(status.Stations, instance20.status()...)
	}

	return status
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
	if cfg.Port != 0 {
		port = cfg.Port
	}
	if cfg.Port20 != 0 {
		port20 = cfg.Port20
	}
	externalUrl = networkExternalUrl
}

func Instance() *CS {
	once.Do(func() {
		log := util.NewLogger("ocpp")

		server := ws.NewServer()
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

// Instance20 returns the OCPP 2.0.1 CSMS singleton
func Instance20() *CSMS20 {
	once20.Do(func() {
		log := util.NewLogger("ocpp20")

		server := ws.NewServer()
		server.SetCheckOriginHandler(func(r *http.Request) bool { return true })

		dispatcher := ocppj.NewDefaultServerDispatcher(ocppj.NewFIFOQueueMap(0))
		dispatcher.SetTimeout(Timeout)

		endpoint := ocppj.NewServer(server, dispatcher, nil,
			provisioning.Profile,
			authorization.Profile,
			localauth.Profile,
			transactions.Profile,
			remotecontrol.Profile,
			availability.Profile,
			reservation.Profile,
			tariffcost.Profile,
			meter.Profile,
			smartcharging20.Profile,
			firmware.Profile,
			iso15118.Profile,
			diagnostics.Profile,
			display.Profile,
			data.Profile,
			security20.Profile,
		)
		endpoint.SetInvalidMessageHook(func(client ws.Channel, err *ocpp.Error, rawMessage string, parsedFields []any) *ocpp.Error {
			log.ERROR.Printf("%v (%s)", err, rawMessage)
			return nil
		})

		csms := ocpp2.NewCSMS(endpoint, server)

		instance20 = &CSMS20{
			log:  log,
			regs: make(map[string]*registration20),
			CSMS: csms,
		}

		ocppj.SetLogger(instance20)

		csms.SetProvisioningHandler(instance20)
		csms.SetAuthorizationHandler(instance20)
		csms.SetAvailabilityHandler(instance20)
		csms.SetTransactionsHandler(instance20)
		csms.SetMeterHandler(instance20)
		csms.SetSecurityHandler(instance20)
		csms.SetSmartChargingHandler(instance20)
		csms.SetNewChargingStationHandler(instance20.NewChargingStation)
		csms.SetChargingStationDisconnectedHandler(instance20.ChargingStationDisconnected)

		go instance20.errorHandler(csms.Errors())
		go csms.Start(port20, "/{ws}")

		// wait for server to start
		for range time.Tick(10 * time.Millisecond) {
			if dispatcher.IsRunning() {
				break
			}
		}
	})

	return instance20
}
