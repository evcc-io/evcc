package charger

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/charger/ocpp20"
	ocpp2 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/remotecontrol"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
	"github.com/stretchr/testify/suite"
)

const (
	ocpp20TestUrl            = "ws://localhost:8886"
	ocpp20TestConnectTimeout = 10 * time.Second
)

func TestOcpp20(t *testing.T) {
	suite.Run(t, new(ocpp20TestSuite))
}

type ocpp20TestSuite struct {
	suite.Suite
}

func (suite *ocpp20TestSuite) SetupSuite() {
	ocpp.Timeout = 5 * time.Second

	// boot the OCPP 2.0.1 CSMS singleton listening on :8886
	suite.NotNil(ocpp20.Instance())
}

// startStation spins up a fake OCPP 2.0.1 charging station that connects
// to the in-process CSMS over a real WebSocket. The handler responds to
// CSMS-initiated GetVariables / SetChargingProfile / TriggerMessage calls.
func (suite *ocpp20TestSuite) startStation(id string, phaseSwitchOK bool) (ocpp2.ChargingStation, *chargingStation20Handler) {
	handler := &chargingStation20Handler{
		triggerC:      make(chan remotecontrol.MessageTrigger, 4),
		phaseSwitchOK: phaseSwitchOK,
	}

	station := ocpp2.NewChargingStation(id, nil, nil)
	station.SetProvisioningHandler(handler)
	station.SetAvailabilityHandler(handler)
	station.SetTransactionsHandler(handler)
	station.SetRemoteControlHandler(handler)
	station.SetReservationHandler(handler)
	station.SetLocalAuthListHandler(handler)
	station.SetSecurityHandler(handler)
	station.SetSmartChargingHandler(handler)
	station.SetFirmwareHandler(handler)
	station.SetISO15118Handler(handler)
	station.SetDiagnosticsHandler(handler)
	station.SetDisplayHandler(handler)
	station.SetDataHandler(handler)
	station.SetTariffCostHandler(handler)

	// auto-respond to CSMS-triggered BootNotification
	go func() {
		for msg := range handler.triggerC {
			if msg == provisioning.BootNotificationFeatureName {
				_, _ = station.BootNotification(provisioning.BootReasonPowerUp, "model", "vendor")
			}
		}
	}()

	return station, handler
}

func (suite *ocpp20TestSuite) TestE2E() {
	// 1. station-side WebSocket connect and initial Boot/Status flow
	station, _ := suite.startStation("station-e2e", true)
	suite.Require().NoError(station.Start(ocpp20TestUrl))
	suite.Require().True(station.IsConnected())

	// volunteer a BootNotification — the CSMS expects it before accepting other messages
	_, err := station.BootNotification(provisioning.BootReasonPowerUp, "model", "vendor")
	suite.Require().NoError(err)

	// 2. evcc-side OCPP 2.0.1 charger creation runs concurrently:
	//    NewOCPP20 calls cp.Setup() → GetVariables, then blocks on evse.Initialized()
	//    waiting for the first StatusNotification that targets this EVSE.
	type result struct {
		c   *OCPP20
		err error
	}
	done := make(chan result, 1)
	go func() {
		c, err := NewOCPP20(suite.T().Context(),
			"station-e2e", 1, 1, "",
			10*time.Second,
			false, false, false, false,
			ocpp20TestConnectTimeout)
		done <- result{c, err}
	}()

	// give NewOCPP20 a moment to register the EVSE listener, then deliver status.
	time.Sleep(200 * time.Millisecond)
	_, err = station.StatusNotification(types.NewDateTime(time.Now()), availability.ConnectorStatusAvailable, 1, 1)
	suite.Require().NoError(err)

	res := <-done
	suite.Require().NoError(res.err)
	suite.Require().NotNil(res.c)
	c := res.c

	// 3. status maps to api.StatusA when no transaction
	status, err := c.Status()
	suite.Require().NoError(err)
	suite.Equal(api.StatusA, status)

	// 4. simulate plug-in: status switches to Occupied, transaction starts charging
	_, err = station.StatusNotification(types.NewDateTime(time.Now()), availability.ConnectorStatusOccupied, 1, 1)
	suite.Require().NoError(err)

	connectorID := 1
	_, err = station.TransactionEvent(
		transactions.TransactionEventStarted,
		types.NewDateTime(time.Now()),
		transactions.TriggerReasonCablePluggedIn,
		1,
		transactions.Transaction{
			TransactionID: "txn-e2e-1",
			ChargingState: transactions.ChargingStateCharging,
		},
		func(req *transactions.TransactionEventRequest) {
			req.Evse = &types.EVSE{ID: 1, ConnectorID: &connectorID}
			req.MeterValue = []types.MeterValue{
				{
					Timestamp: *types.NewDateTime(time.Now()),
					SampledValue: []types.SampledValue{
						{Measurand: types.MeasurandPowerActiveImport, Value: 4242},
						{Measurand: types.MeasurandEnergyActiveImportRegister, Value: 1500},
					},
				},
			}
		},
	)
	suite.Require().NoError(err)

	suite.Eventually(func() bool {
		st, err := c.Status()
		return err == nil && st == api.StatusC
	}, 2*time.Second, 50*time.Millisecond, "expected StatusC after charging transaction event")

	// 5. evcc reads CurrentPower / TotalEnergy from the meter samples
	power, err := c.evse.CurrentPower()
	suite.Require().NoError(err)
	suite.InDelta(4242.0, power, 0.001)

	energy, err := c.evse.TotalEnergy()
	suite.Require().NoError(err)
	suite.InDelta(1.5, energy, 0.001) // 1500 Wh → 1.5 kWh

	// 6. evcc → station: MaxCurrent triggers SetChargingProfile (handler accepts)
	suite.Require().NoError(c.MaxCurrent(16))

	// 7. transaction ends — evse clears it
	_, err = station.TransactionEvent(
		transactions.TransactionEventEnded,
		types.NewDateTime(time.Now()),
		transactions.TriggerReasonEVCommunicationLost,
		2,
		transactions.Transaction{
			TransactionID: "txn-e2e-1",
		},
	)
	suite.Require().NoError(err)

	// 8. clean disconnect
	station.Stop()
	suite.Eventually(func() bool { return !station.IsConnected() }, time.Second, 20*time.Millisecond)
}
