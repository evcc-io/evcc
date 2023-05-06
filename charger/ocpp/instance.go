package ocpp

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocppj"
)

var (
	once     sync.Once
	instance *CS
)

func Instance() *CS {
	once.Do(func() {
		cs := ocpp16.NewCentralSystem(nil, nil)

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

		time.Sleep(time.Second)
	})

	return instance
}
