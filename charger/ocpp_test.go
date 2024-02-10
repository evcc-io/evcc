package charger

import (
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/suite"
)

const (
	ocppTestUrl            = "ws://localhost:8887"
	ocppTestConnectTimeout = 10 * time.Second
	ocppTestTimeout        = 3 * time.Second
)

func TestOcpp(t *testing.T) {
	suite.Run(t, new(ocppTestSuite))
}

type ocppTestSuite struct {
	suite.Suite
	clock *clock.Mock
}

func (suite *ocppTestSuite) SetupSuite() {
	suite.clock = clock.NewMock()
	suite.NotNil(ocpp.Instance())
}

func (suite *ocppTestSuite) startChargePoint(id string, connectorId int) ocpp16.ChargePoint {
	// set a handler for all callback functions
	handler := &ChargePointHandler{
		triggerC: make(chan remotetrigger.MessageTrigger, 1),
	}

	// create charge point with handler
	cp := ocpp16.NewChargePoint(id, nil, nil)
	cp.SetCoreHandler(handler)
	cp.SetRemoteTriggerHandler(handler)

	// let cs handle the trigger messages
	go func() {
		for msg := range handler.triggerC {
			suite.handleTrigger(cp, connectorId, msg)
		}
	}()

	return cp
}

func (suite *ocppTestSuite) handleTrigger(cp ocpp16.ChargePoint, connectorId int, msg remotetrigger.MessageTrigger) {
	switch msg {
	case core.BootNotificationFeatureName:
		if res, err := cp.BootNotification("demo", "evcc"); err != nil {
			suite.T().Log("BootNotification:", err)
		} else {
			suite.T().Log("BootNotification:", res)
		}

	case core.StatusNotificationFeatureName:
		if res, err := cp.StatusNotification(connectorId, core.NoError, core.ChargePointStatusAvailable); err != nil {
			suite.T().Log("StatusNotification:", err)
		} else {
			suite.T().Log("StatusNotification:", res)
		}

	case core.MeterValuesFeatureName:
		if res, err := cp.MeterValues(connectorId, []types.MeterValue{
			{
				Timestamp: types.NewDateTime(suite.clock.Now()),
				SampledValue: []types.SampledValue{
					{Measurand: types.MeasurandPowerActiveImport, Value: "1000"},
					{Measurand: types.MeasurandEnergyActiveImportRegister, Value: "1.2", Unit: "kWh"},
				},
			},
		}); err != nil {
			suite.T().Log("MeterValues:", err)
		} else {
			suite.T().Log("MeterValues:", res)
		}

	default:
		suite.T().Log(msg)
	}
}

func (suite *ocppTestSuite) TestConnect() {
	// 1st charge point- remote
	cp1 := suite.startChargePoint("test-1", 1)
	suite.Require().NoError(cp1.Start(ocppTestUrl))
	suite.Require().True(cp1.IsConnected())

	// 1st charge point- local
	c1, err := NewOCPP("test-1", 1, "", "", 0, false, false, ocppTestConnectTimeout, ocppTestTimeout, "A")
	suite.Require().NoError(err)

	// status and meter values
	{
		suite.clock.Add(ocppTestTimeout)
		c1.conn.TestClock(suite.clock)

		// status
		_, err = c1.Status()
		suite.Require().NoError(err)

		// power
		f, err := c1.currentPower()
		suite.Require().NoError(err)
		suite.Equal(1e3, f)

		// energy
		f, err = c1.totalEnergy()
		suite.Require().NoError(err)
		suite.Equal(1.2, f)
	}

	// takeover
	{
		expectedTxn := 99

		_, err := cp1.StopTransaction(0, types.NewDateTime(suite.clock.Now()), expectedTxn)
		suite.Require().Error(err)

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
	cp2 := suite.startChargePoint("test-2", 1)
	suite.Require().NoError(cp2.Start(ocppTestUrl))
	suite.Require().True(cp2.IsConnected())

	// 2nd charge point - local
	c2, err := NewOCPP("test-2", 1, "", "", 0, false, false, ocppTestConnectTimeout, ocppTestTimeout, "A")
	suite.Require().NoError(err)

	{
		suite.clock.Add(ocppTestTimeout)
		c2.conn.TestClock(suite.clock)

		// status
		_, err = c2.Status()
		suite.Require().NoError(err)
	}

	// error on unconfigured 2nd charge point
	cp3 := suite.startChargePoint("unconfigured", 1)
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
