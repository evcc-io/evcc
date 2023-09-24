package charger

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
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
	ocppTestConnector      = 1
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

func (suite *ocppTestSuite) startChargePoint(id string) ocpp16.ChargePoint {
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
			suite.handleTrigger(cp, msg)
		}
	}()

	return cp
}

func (suite *ocppTestSuite) handleTrigger(cp ocpp16.ChargePoint, msg remotetrigger.MessageTrigger) {
	switch msg {
	case core.BootNotificationFeatureName:
		if res, err := cp.BootNotification("demo", "evcc"); err != nil {
			suite.T().Log("BootNotification:", err)
		} else {
			suite.T().Log("BootNotification:", res)
		}

	case core.StatusNotificationFeatureName:
		if res, err := cp.StatusNotification(ocppTestConnector, core.NoError, core.ChargePointStatusAvailable); err != nil {
			suite.T().Log("StatusNotification:", err)
		} else {
			suite.T().Log("StatusNotification:", res)
		}

	case core.MeterValuesFeatureName:
		if res, err := cp.MeterValues(1, []types.MeterValue{
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
	// start cp client
	cp := suite.startChargePoint("test")
	suite.NoError(cp.Start(ocppTestUrl))
	suite.True(cp.IsConnected())

	// start cp server
	c, err := NewOCPP("test", ocppTestConnector, "", "", 0, false, false, ocppTestConnectTimeout, ocppTestTimeout)
	if err != nil {
		suite.NoError(err)
		return
	}

	suite.clock.Add(ocppTestTimeout)
	c.cp.TestClock(suite.clock)

	// status
	_, err = c.Status()
	suite.NoError(err)

	// power
	f, err := c.currentPower()
	suite.NoError(err)
	suite.Equal(1e3, f)

	// energy
	f, err = c.totalEnergy()
	suite.NoError(err)
	suite.Equal(1.2, f)

	// 2nd charge point
	cp2 := suite.startChargePoint("test2")
	suite.NoError(cp2.Start(ocppTestUrl))
	suite.True(cp2.IsConnected())

	// error on unconfigured 2nd charge point
	_, err = cp2.BootNotification("demo", "evcc")
	suite.Error(err)
}
