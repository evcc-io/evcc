package ocpp20

import (
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	ocppgo "github.com/lorenzodonini/ocpp-go/ocpp"
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
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/security"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/tariffcost"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

var (
	once     sync.Once
	instance *CSMS
)

func init() {
	ocpp.RegisterStatusHook20(func() []ocpp.StationInfo {
		if instance == nil {
			return nil
		}
		return instance.status()
	})
}

// Instance returns the OCPP 2.0.1 CSMS singleton
func Instance() *CSMS {
	once.Do(func() {
		log := util.NewLogger("ocpp20")

		server := ws.NewServer()
		server.SetCheckOriginHandler(func(r *http.Request) bool { return true })

		dispatcher := ocppj.NewDefaultServerDispatcher(ocppj.NewFIFOQueueMap(0))
		dispatcher.SetTimeout(ocpp.Timeout)

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
			smartcharging.Profile,
			firmware.Profile,
			iso15118.Profile,
			diagnostics.Profile,
			display.Profile,
			data.Profile,
			security.Profile,
		)
		endpoint.SetInvalidMessageHook(func(client ws.Channel, err *ocppgo.Error, rawMessage string, parsedFields []any) *ocppgo.Error {
			log.ERROR.Printf("%v (%s)", err, rawMessage)
			return nil
		})

		csms := ocpp2.NewCSMS(endpoint, server)

		instance = &CSMS{
			log:  log,
			regs: make(map[string]*registration),
			CSMS: csms,
		}

		ocppj.SetLogger(instance)

		csms.SetProvisioningHandler(instance)
		csms.SetAuthorizationHandler(instance)
		csms.SetAvailabilityHandler(instance)
		csms.SetTransactionsHandler(instance)
		csms.SetMeterHandler(instance)
		csms.SetSecurityHandler(instance)
		csms.SetSmartChargingHandler(instance)
		csms.SetNewChargingStationHandler(instance.NewChargingStation)
		csms.SetChargingStationDisconnectedHandler(instance.ChargingStationDisconnected)

		go instance.errorHandler(csms.Errors())
		go csms.Start(ocpp.Port20(), "/{ws}")

		// wait for server to start
		for range time.Tick(10 * time.Millisecond) {
			if dispatcher.IsRunning() {
				break
			}
		}
	})

	return instance
}
