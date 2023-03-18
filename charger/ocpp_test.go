package charger

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/charger/ocpp"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/stretchr/testify/suite"
)

const (
	ocppTestConnectTimeout = 10 * time.Second
	ocppTestTimeout        = 5 * time.Second
	ocppTestUrl            = "ws://localhost:8887"
)

func TestOcpp(t *testing.T) {
	suite.Run(t, new(ocppTestSuite))
}

type ocppTestSuite struct {
	suite.Suite
	cp ocpp16.ChargePoint
}

func (suite *ocppTestSuite) SetupSuite() {
	// setup cs
	suite.NotNil(ocpp.Instance())

	// setup cp
	cp := ocpp16.NewChargePoint("test", nil, nil)

	// set a handler for all callback functions
	handler := &ChargePointHandler{triggerC: make(chan remotetrigger.MessageTrigger, 1)}
	cp.SetCoreHandler(handler)
	cp.SetRemoteTriggerHandler(handler)

	suite.cp = cp
}

func (suite *ocppTestSuite) TestConnect() {
	// start cp client
	suite.NoError(suite.cp.Start(ocppTestUrl))

	// start cp server
	c, err := NewOCPP("test", 0, "", "", 0, false, false, ocppTestConnectTimeout, ocppTestTimeout)
	suite.NoError(err)

	_ = c
}
