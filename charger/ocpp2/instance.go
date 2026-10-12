package ocpp2

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	ocpp2 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/authorization"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/diagnostics"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/meter"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ws"
)

type Config struct {
	Port int `json:"port"`
}

var (
	instance2   *CS2
	started2    func() error
	port2       = 8888
	boundPort2  int
	externalUrl2 string
)

func Port2() int { return boundPort2 }

func ExternalUrl2() string {
	if externalUrl2 == "" {
		return ""
	}
	u, err := url.Parse(externalUrl2)
	if err != nil {
		return ""
	}
	u.Scheme = "ws"
	u.Host = fmt.Sprintf("%s:%d", strings.Split(u.Host, ":")[0], port2)
	return u.String()
}

func NewServer2(cfg Config, networkExternalUrl string) {
	port2 = cfg.Port
	externalUrl2 = networkExternalUrl

	log := util.NewLogger("ocpp2")

	// ws.Server erstellen und konfigurieren
	server := ws.NewServer()
	server.SetCheckOriginHandler(func(r *http.Request) bool { return true })
	timeouts := ws.NewServerTimeoutConfig()
	timeouts.PingWait = pingWait
	server.SetTimeoutConfig(timeouts)

	// KORRIGIERT: NewCSMS(endpoint *ocppj.Server, server ws.Server)
	// nil als endpoint → Bibliothek erstellt Standard-Endpoint mit allen Profilen
	csms := ocpp2.NewCSMS(nil, server)

	inst := &CS2{
		CSMS:   csms,
		log:    log,
		regs:   make(map[string]*registration2),
		server: server, // ws.Server für Addr()-Check in listen()
	}
	inst.txnId.Store(time.Now().UTC().Unix())

	// Handler registrieren
	csms.SetProvisioningHandler(inst)
	csms.SetAvailabilityHandler(inst)
	csms.SetTransactionsHandler(inst)
	csms.SetMeterHandler(inst)
	csms.SetAuthorizationHandler(inst)
	csms.SetSmartChargingHandler(inst)
	csms.SetDiagnosticsHandler(inst)
	csms.SetFirmwareHandler(inst)

	csms.SetNewChargingStationHandler(inst.NewChargingStation)
	csms.SetChargingStationDisconnectedHandler(inst.ChargingStationDisconnected)

	started2 = sync.OnceValue(inst.listen)
	instance2 = inst
}

func (cs *CS2) listen() error {
	// Fehler-Channel im Hintergrund loggen
	go cs.errorHandler(cs.CSMS.Errors())

	// CSMS starten
	go cs.CSMS.Start(port2, "/{ws}")

	// Warten bis Port gebunden (max. 10 Sekunden)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	timeout := time.After(10 * time.Second)

	for cs.server.Addr() == nil {
		select {
		case <-tick.C:
		case <-timeout:
			return errors.New("ocpp2: timeout waiting for server to bind")
		}
	}

	boundPort2 = cs.server.Addr().Port
	cs.log.INFO.Printf("OCPP 2.0.1 CSMS listening on port %d", boundPort2)
	return nil
}

func Instance2() (*CS2, error) {
	if instance2 == nil {
		return nil, errors.New("ocpp2 not configured")
	}
	return instance2, started2()
}

// Compile-time checks
var (
	_ provisioning.CSMSHandler  = (*CS2)(nil)
	_ availability.CSMSHandler  = (*CS2)(nil)
	_ transactions.CSMSHandler  = (*CS2)(nil)
	_ meter.CSMSHandler         = (*CS2)(nil)
	_ authorization.CSMSHandler = (*CS2)(nil)
	_ smartcharging.CSMSHandler = (*CS2)(nil)
	_ diagnostics.CSMSHandler   = (*CS2)(nil)
	_ firmware.CSMSHandler      = (*CS2)(nil)
)