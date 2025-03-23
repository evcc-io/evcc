package charger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	ocppapi "github.com/lorenzodonini/ocpp-go/ocpp"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
	"github.com/stretchr/testify/suite"
)

const (
	ocppTestUrl            = "ws://localhost:8887"
	ocppTestConnectTimeout = 10 * time.Second
)

func TestOcpp(t *testing.T) {
	suite.Run(t, new(ocppTestSuite))
}

type ocppTestSuite struct {
	suite.Suite
	clock *clock.Mock
}

func (suite *ocppTestSuite) SetupSuite() {
	ocpp.Timeout = 5 * time.Second

	// setup cs so we can overwrite logger afterwards
	_ = ocpp.Instance()
	ocppj.SetLogger(&ocppLogger{suite.T()})

	suite.clock = clock.NewMock()
	suite.NotNil(ocpp.Instance())
}

func (suite *ocppTestSuite) startChargePoint(id string, connectorId int) (ocpp16.ChargePoint, *ocppj.Client) {
	// set a handler for all callback functions
	handler := &ChargePointHandler{
		triggerC: make(chan remotetrigger.MessageTrigger, 1),
	}

	// ocppj endpoint with handler
	client := ws.NewClient()
	client.SetRequestedSubProtocol(types.V16Subprotocol)
	dispatcher := ocppj.NewDefaultClientDispatcher(ocppj.NewFIFOClientQueue(0))
	endpoint := ocppj.NewClient(id, client, dispatcher, nil, core.Profile, localauth.Profile, firmware.Profile, reservation.Profile, remotetrigger.Profile, smartcharging.Profile)

	// create charge point with handler
	cp := ocpp16.NewChargePoint(id, endpoint, client)
	cp.SetCoreHandler(handler)
	cp.SetRemoteTriggerHandler(handler)
	cp.SetSmartChargingHandler(handler)

	// let cs handle the trigger messages
	go func() {
		for msg := range handler.triggerC {
			suite.handleTrigger(cp, connectorId, msg)
		}
	}()

	return cp, endpoint
}

func (suite *ocppTestSuite) handleTrigger(cp ocpp16.ChargePoint, connectorId int, msg remotetrigger.MessageTrigger) {
	switch msg {
	case core.BootNotificationFeatureName:
		if _, err := cp.BootNotification("model", "vendor"); err != nil {
			suite.T().Log("BootNotification:", err)
		}

	case core.ChangeAvailabilityFeatureName:
		fallthrough

	case core.StatusNotificationFeatureName:
		if _, err := cp.StatusNotification(connectorId, core.NoError, core.ChargePointStatusCharging); err != nil {
			suite.T().Log("StatusNotification:", err)
		}

	case core.MeterValuesFeatureName:
		if _, err := cp.MeterValues(connectorId, []types.MeterValue{
			{
				Timestamp: types.NewDateTime(suite.clock.Now()),
				SampledValue: []types.SampledValue{
					{Measurand: types.MeasurandPowerActiveImport, Value: "1000"},
					{Measurand: types.MeasurandEnergyActiveImportRegister, Value: "1.2", Unit: "kWh"},
				},
			},
		}); err != nil {
			suite.T().Log("MeterValues:", err)
		}
	}
}

func (suite *ocppTestSuite) TestConnect() {
	// 1st charge point- remote
	cp1, _ := suite.startChargePoint("test-1", 1)
	suite.Require().NoError(cp1.Start(ocppTestUrl))
	suite.Require().True(cp1.IsConnected())

	// 1st charge point- local
	c1, err := NewOCPP(context.TODO(), "test-1", 1, "", "", 0, false, false, true, ocppTestConnectTimeout)
	suite.Require().NoError(err)

	// status and meter values
	{
		suite.clock.Add(ocpp.Timeout)
		c1.conn.TestClock(suite.clock)

		// status
		_, err = c1.Status()
		suite.Require().NoError(err)
	}

	// takeover
	{
		expectedTxn := 99

		// always accept stopping unknown transaction, see https://github.com/evcc-io/evcc/pull/13990
		_, err := cp1.StopTransaction(0, types.NewDateTime(suite.clock.Now()), expectedTxn)
		suite.Require().NoError(err)

		_, err = cp1.MeterValues(1, []types.MeterValue{
			{
				Timestamp: types.NewDateTime(suite.clock.Now()),
				SampledValue: []types.SampledValue{
					{Measurand: types.MeasurandPowerActiveImport, Value: "1000"},
				},
			},
		}, func(request *core.MeterValuesRequest) {
			request.TransactionId = &expectedTxn
		})
		suite.Require().NoError(err)

		conn1 := c1.Connector()
		txnId, err := conn1.TransactionID()
		suite.Require().NoError(err)
		suite.Equal(expectedTxn, txnId)

		res, err := cp1.StopTransaction(0, types.NewDateTime(suite.clock.Now()), expectedTxn)
		suite.Require().NoError(err)
		suite.Equal(types.AuthorizationStatusAccepted, res.IdTagInfo.Status)
	}

	// 2nd charge point - remote
	cp2, _ := suite.startChargePoint("test-2", 1)
	suite.Require().NoError(cp2.Start(ocppTestUrl))
	suite.Require().True(cp2.IsConnected())

	// 2nd charge point - local
	c2, err := NewOCPP(context.TODO(), "test-2", 1, "", "", 0, false, false, true, ocppTestConnectTimeout)
	suite.Require().NoError(err)

	{
		suite.clock.Add(ocpp.Timeout)
		c2.conn.TestClock(suite.clock)

		// status
		_, err = c2.Status()
		suite.Require().NoError(err)
	}

	// error on unconfigured 2nd charge point
	cp3, _ := suite.startChargePoint("unconfigured", 1)
	_, err = cp3.BootNotification("model", "vendor")
	suite.Require().Error(err)

	// disconnect charge point
	cp2.Stop()
	suite.Require().False(cp2.IsConnected())

	t := time.NewTimer(100 * time.Millisecond)
WAIT_DISCONNECT:
	for {
		select {
		case <-t.C:
			suite.Fail("disconnect timeout")
		case <-time.After(10 * time.Millisecond):
			if _, err := c2.Status(); errors.Is(err, api.ErrTimeout) {
				break WAIT_DISCONNECT
			}
		}
	}
}

func (suite *ocppTestSuite) TestAutoStart() {
	// 1st charge point- remote
	cp1, _ := suite.startChargePoint("test-3", 1)
	suite.Require().NoError(cp1.Start(ocppTestUrl))
	suite.Require().True(cp1.IsConnected())

	// 1st charge point- local
	c1, err := NewOCPP(context.TODO(), "test-3", 1, "", "", 0, false, false, false, ocppTestConnectTimeout)
	suite.Require().NoError(err)

	// status and meter values
	{
		suite.clock.Add(ocpp.Timeout)
		c1.conn.TestClock(suite.clock)
	}

	// acquire
	{
		expectedIdTag := "tag"

		// always accept stopping unknown transaction, see https://github.com/evcc-io/evcc/pull/13990
		_, err := cp1.StartTransaction(1, expectedIdTag, 0, types.NewDateTime(suite.clock.Now()))
		suite.Require().NoError(err)

		id, err := c1.Identify()
		suite.Require().NoError(err)
		suite.Require().Equal(expectedIdTag, id)

		conn1 := c1.Connector()
		_, err = conn1.TransactionID()
		suite.Require().NoError(err)
	}

	err = c1.Enable(true)
	suite.Require().NoError(err)

	err = c1.Enable(false)
	suite.Require().NoError(err)
}

func (suite *ocppTestSuite) TestTimeout() {
	// 1st charge point- remote
	cp1, ocppjClient := suite.startChargePoint("test-4", 1)
	suite.Require().NoError(cp1.Start(ocppTestUrl))
	suite.Require().True(cp1.IsConnected())

	handler := ocppjClient.GetRequestHandler()
	ocppjClient.SetRequestHandler(func(request ocppapi.Request, requestId string, action string) {
		if action != core.ChangeAvailabilityFeatureName {
			handler(request, requestId, action)
		}
	})

	// 1st charge point- local
	_, err := NewOCPP(context.TODO(), "test-4", 1, "", "", 0, false, false, false, ocppTestConnectTimeout)

	suite.Require().NoError(err)
}
