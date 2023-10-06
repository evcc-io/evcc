package ocpp

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

var (
	once     sync.Once
	instance *CS
)

func Instance() *CS {
	once.Do(func() {
		timeoutConfig := ws.NewServerTimeoutConfig()
		timeoutConfig.PingWait = 90 * time.Second

		server := ws.NewServer()
		server.SetTimeoutConfig(timeoutConfig)

		dispatcher := ocppj.NewDefaultServerDispatcher(ocppj.NewFIFOQueueMap(0))
		dispatcher.SetTimeout(time.Minute)

		endpoint := ocppj.NewServer(server, dispatcher, nil, core.Profile, localauth.Profile, firmware.Profile, reservation.Profile, remotetrigger.Profile, smartcharging.Profile)

		cs := ocpp16.NewCentralSystem(endpoint, server)

		instance = &CS{
			log:           util.NewLogger("ocpp"),
			cps:           make(map[string]*CP),
			CentralSystem: cs,
		}

		ocppj.SetLogger(instance)

		cs.SetCoreHandler(instance)
		cs.SetNewChargePointHandler(instance.NewChargePoint)
		cs.SetChargePointDisconnectedHandler(instance.ChargePointDisconnected)
		cs.SetFirmwareManagementHandler(instance)

		go instance.errorHandler(cs.Errors())
		go cs.Start(8887, "/{ws}")

		// wait for server to start
		for range time.Tick(10 * time.Millisecond) {
			if dispatcher.IsRunning() {
				break
			}
		}
	})

	return instance
}
