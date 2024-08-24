package ocpp

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
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

func (suite *connTestSuite) SetupSuite() {
	suite.cp = NewChargePoint(util.NewLogger("foo"), "abc")
	suite.conn, _ = NewConnector(util.NewLogger("foo"), 1, suite.cp, time.Minute)

	suite.clock = clock.NewMock()
	suite.conn.clock = suite.clock
	suite.conn.cp.connected = true

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
	} {
		suite.conn.measurements[m] = types.SampledValue{Value: "1"}
	}
}

func (suite *connTestSuite) TestConnectorMeasurementsNoTxn() {
	// connected, no txn, no meter update since 1 hour
	suite.clock.Add(time.Hour)

	// intentionally no error
	_, err := suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	_, _, _, err = suite.conn.Currents()
	suite.NoError(err, "Currents")
	_, err = suite.conn.TotalEnergy()

	// intentionally no error ???
	suite.Equal(api.ErrTimeout, err, "TotalEnergy")
	_, err = suite.conn.GetMaxCurrent()
	suite.Equal(api.ErrTimeout, err, "GetMaxCurrent")
	_, _, _, err = suite.conn.Voltages()
	suite.Equal(api.ErrTimeout, err, "Voltages")
}

func (suite *connTestSuite) TestConnectorMeasurementsRunningTxnOutdated() {
	// connected, running txn, no meter update since 1 hour
	suite.clock.Add(time.Hour)
	suite.conn.txnId = 1

	_, err := suite.conn.CurrentPower()
	suite.Equal(api.ErrTimeout, err, "CurrentPower")
	_, err = suite.conn.TotalEnergy()
	suite.Equal(api.ErrTimeout, err, "TotalEnergy")
	_, err = suite.conn.GetMaxCurrent()
	suite.Equal(api.ErrTimeout, err, "GetMaxCurrent")
	_, _, _, err = suite.conn.Currents()
	suite.Equal(api.ErrTimeout, err, "Currents")
	_, _, _, err = suite.conn.Voltages()
	suite.Equal(api.ErrTimeout, err, "Voltages")
}

func (suite *connTestSuite) TestConnectorMeasurementsRunningTxn() {
	// connected, running txn, no meter update since 1 hour
	suite.clock.Add(time.Hour)
	suite.conn.meterUpdated = suite.clock.Now()
	suite.conn.txnId = 1

	_, err := suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	_, err = suite.conn.TotalEnergy()
	suite.NoError(err, "TotalEnergy")
	_, err = suite.conn.GetMaxCurrent()
	suite.NoError(err, "GetMaxCurrent")
	_, _, _, err = suite.conn.Currents()
	suite.NoError(err, "Currents")
	_, _, _, err = suite.conn.Voltages()
	suite.NoError(err, "Voltages")
}
