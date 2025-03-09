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

func (suite *connTestSuite) SetupTest() {
	// setup instance
	Instance()
	suite.cp = NewChargePoint(util.NewLogger("foo"), "abc")
	suite.conn, _ = NewConnector(util.NewLogger("foo"), 1, suite.cp, "", Timeout)

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
