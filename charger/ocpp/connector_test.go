package ocpp

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/suite"
)

func TestConnector(t *testing.T) {
	suite.Run(t, new(connTestSuite))
}

type connTestSuite struct {
	suite.Suite
	cp    *CP
	conn  *Connector
	clock *clock.Mock
}

func (suite *connTestSuite) SetupTest() {
	// setup instance
	Instance()
	suite.cp = NewChargePoint(util.NewLogger("foo"), "abc")
	suite.conn, _ = NewConnector(suite.T().Context(), util.NewLogger("foo"), 1, suite.cp, "", Timeout)

	suite.clock = clock.NewMock()
	suite.conn.clock = suite.clock
	suite.conn.cp.connected = true
}

func (suite *connTestSuite) addMeasurements() {
	for _, m := range []types.Measurand{
		types.MeasurandPowerActiveImport,
		types.MeasurandEnergyActiveImportRegister,
		types.MeasurandCurrentImport + ".L1",
		types.MeasurandCurrentImport + ".L2",
		types.MeasurandCurrentImport + ".L3",
		types.MeasurandVoltage + ".L1-N",
		types.MeasurandVoltage + ".L2-N",
		types.MeasurandVoltage + ".L3-N",
		types.MeasurandCurrentOffered,
		types.MeasurandSoC,
	} {
		suite.conn.measurements[m] = types.SampledValue{Value: "1"}
	}
}

func (suite *connTestSuite) TestConnectorNoMeasurements() {
	// connected, no txn, no meter update since 1 hour
	suite.clock.Add(time.Hour)

	// intentionally no error
	res, err := suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	suite.Equal(res, 0.0, "CurrentPower")

	res1, res2, res3, err := suite.conn.Currents()
	suite.NoError(err, "Currents")
	suite.Equal(res1, 0.0)
	suite.Equal(res2, 0.0)
	suite.Equal(res3, 0.0)

	_, err = suite.conn.GetMaxCurrent()
	suite.Equal(api.ErrTimeout, err, "GetMaxCurrent")

	// api.ErrNotAvailable
	_, err = suite.conn.TotalEnergy()
	suite.Equal(api.ErrNotAvailable, err, "TotalEnergy")
	_, err = suite.conn.Soc()
	suite.Equal(api.ErrNotAvailable, err, "Soc")
	_, _, _, err = suite.conn.Voltages()
	suite.Equal(api.ErrNotAvailable, err, "Voltages")
}

func (suite *connTestSuite) TestConnectorMeasurementsNoTxn() {
	// connected, no txn, no meter update since 1 hour
	suite.addMeasurements()
	suite.clock.Add(time.Hour)

	// intentionally no error
	res, err := suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	suite.Equal(res, 0.0, "CurrentPower")

	res1, res2, res3, err := suite.conn.Currents()
	suite.NoError(err, "Currents")
	suite.Equal(res1, 0.0)
	suite.Equal(res2, 0.0)
	suite.Equal(res3, 0.0)

	// api.ErrTimeout
	_, err = suite.conn.GetMaxCurrent()
	suite.Equal(api.ErrTimeout, err, "GetMaxCurrent")

	// keep old values
	res, err = suite.conn.TotalEnergy()
	suite.NoError(err, "TotalEnergy")
	suite.Equal(res, 0.001, "TotalEnergy")
	res, err = suite.conn.Soc()
	suite.NoError(err, "Soc")
	suite.Equal(res, 1.0, "Soc")
	res1, res2, res3, err = suite.conn.Voltages()
	suite.NoError(err, "Voltages")
	suite.Equal(res1, 1.0, "Voltages")
	suite.Equal(res2, 1.0, "Voltages")
	suite.Equal(res3, 1.0, "Voltages")
}

func (suite *connTestSuite) TestConnectorMeasurementsRunningTxnOutdated() {
	// connected, running txn, no meter update since 1 hour
	suite.addMeasurements()
	suite.clock.Add(time.Hour)
	suite.conn.txnId = 1

	_, err := suite.conn.CurrentPower()
	suite.Equal(api.ErrTimeout, err, "CurrentPower")
	_, err = suite.conn.TotalEnergy()
	suite.Equal(api.ErrTimeout, err, "TotalEnergy")
	_, err = suite.conn.GetMaxCurrent()
	suite.Equal(api.ErrTimeout, err, "GetMaxCurrent")
	_, err = suite.conn.Soc()
	suite.Equal(api.ErrTimeout, err, "Soc")
	_, _, _, err = suite.conn.Currents()
	suite.Equal(api.ErrTimeout, err, "Currents")
	_, _, _, err = suite.conn.Voltages()
	suite.Equal(api.ErrTimeout, err, "Voltages")
}

func (suite *connTestSuite) TestConnectorMeasurementsRunningTxn() {
	// connected, running txn, no meter update since 1 hour
	suite.addMeasurements()
	suite.clock.Add(time.Hour)
	suite.conn.meterUpdated = suite.clock.Now()
	suite.conn.txnId = 1

	_, err := suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	_, err = suite.conn.TotalEnergy()
	suite.NoError(err, "TotalEnergy")
	_, err = suite.conn.GetMaxCurrent()
	suite.NoError(err, "GetMaxCurrent")
	_, err = suite.conn.Soc()
	suite.NoError(err, "Soc")
	_, _, _, err = suite.conn.Currents()
	suite.NoError(err, "Currents")
	_, _, _, err = suite.conn.Voltages()
	suite.NoError(err, "Voltages")
}

func (suite *connTestSuite) TestStatusNotificationPreparingNonBlocking() {
	// set remoteIdTag to simulate remoteStart=true
	suite.conn.remoteIdTag = "evcc"
	suite.conn.status = nil

	// first StatusNotification(Preparing) should set status and trigger remote start asynchronously
	req := &core.StatusNotificationRequest{
		ConnectorId: 1,
		ErrorCode:   core.NoError,
		Status:      core.ChargePointStatusPreparing,
	}

	// OnStatusNotification must return immediately (non-blocking), even though
	// RemoteStartTransactionRequest would block if called synchronously
	done := make(chan struct{})
	go func() {
		_, err := suite.conn.OnStatusNotification(req)
		suite.NoError(err)
		close(done)
	}()

	select {
	case <-done:
		// handler returned promptly
	case <-time.After(time.Second):
		suite.Fail("OnStatusNotification blocked for too long")
	}

	// status should be Preparing and waiting for auth
	suite.True(suite.conn.NeedsAuthentication(), "should need authentication after Preparing status")
}

func (suite *connTestSuite) TestRemoteStartAfterReconnect() {
	// set remoteIdTag to simulate remoteStart=true
	suite.conn.remoteIdTag = "evcc"
	suite.conn.status = nil

	// 1st connection: StatusNotification(Preparing) -> isWaitingForAuth should be true
	req := &core.StatusNotificationRequest{
		ConnectorId: 1,
		ErrorCode:   core.NoError,
		Status:      core.ChargePointStatusPreparing,
	}
	_, err := suite.conn.OnStatusNotification(req)
	suite.NoError(err)
	suite.True(suite.conn.NeedsAuthentication(), "should need authentication")

	// simulate StartTransaction (charger accepted remote start)
	startReq := &core.StartTransactionRequest{
		ConnectorId: 1,
		IdTag:       "evcc",
		MeterStart:  0,
		Timestamp:   types.NewDateTime(suite.clock.Now()),
	}
	_, err = suite.conn.OnStartTransaction(startReq)
	suite.NoError(err)

	// no longer waiting for auth (txnId is set)
	suite.False(suite.conn.NeedsAuthentication(), "should not need authentication during transaction")
	suite.Equal("evcc", suite.conn.IdTag())

	// simulate StopTransaction (vehicle unplugged)
	_, err = suite.conn.OnStopTransaction(nil)
	suite.NoError(err)

	txnId, err := suite.conn.TransactionID()
	suite.NoError(err)
	suite.Equal(0, txnId, "txnId should be reset after StopTransaction")

	// simulate StatusNotification(Available) during disconnect
	availReq := &core.StatusNotificationRequest{
		ConnectorId: 1,
		ErrorCode:   core.NoError,
		Status:      core.ChargePointStatusAvailable,
	}
	_, err = suite.conn.OnStatusNotification(availReq)
	suite.NoError(err)
	suite.False(suite.conn.NeedsAuthentication(), "should not need authentication when Available")

	// 2nd connection: StatusNotification(Preparing) should again trigger waiting for auth
	req2 := &core.StatusNotificationRequest{
		ConnectorId: 1,
		ErrorCode:   core.NoError,
		Status:      core.ChargePointStatusPreparing,
	}
	_, err = suite.conn.OnStatusNotification(req2)
	suite.NoError(err)
	suite.True(suite.conn.NeedsAuthentication(), "should need authentication again after reconnect")
}

func (suite *connTestSuite) TestOnStopTransactionResetsReportedPower() {
	suite.conn.meterUpdated = suite.clock.Now()

	// Set some power
	suite.conn.measurements[types.MeasurandPowerActiveImport+".L1-N"] = types.SampledValue{Value: "1"}
	suite.conn.measurements[types.MeasurandPowerActiveImport+".L2-N"] = types.SampledValue{Value: "1"}
	suite.conn.measurements[types.MeasurandPowerActiveImport+".L3-N"] = types.SampledValue{Value: "1"}

	res, err := suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	suite.Equal(res, 3.0, "CurrentPower")

	// set powers to zero
	suite.conn.OnStopTransaction(nil)

	res, err = suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	suite.Equal(res, 0.0, "CurrentPower")
}
