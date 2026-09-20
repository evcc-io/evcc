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
	suite.cp = NewChargePoint(util.NewLogger("foo"), instance, "abc")
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

func (suite *connTestSuite) TestConnectorEnergyOnlyNoTxn() {
	// connected, no txn, only energy register reported (e.g. Mennekes ACU while idle)
	suite.conn.measurements[types.MeasurandEnergyActiveImportRegister] = types.SampledValue{Value: "1000", Unit: types.UnitOfMeasureWh}
	suite.conn.meterUpdated = suite.clock.Now()

	// no power measurand but no running txn: report zero instead of ErrNotAvailable
	res, err := suite.conn.CurrentPower()
	suite.NoError(err, "CurrentPower")
	suite.Equal(0.0, res, "CurrentPower")

	// energy is still reported
	res, err = suite.conn.TotalEnergy()
	suite.NoError(err, "TotalEnergy")
	suite.Equal(1.0, res, "TotalEnergy")
}

func (suite *connTestSuite) TestConnectorEnergyOnlyRunningTxn() {
	// connected, running txn, only energy register reported and no power yet
	suite.conn.measurements[types.MeasurandEnergyActiveImportRegister] = types.SampledValue{Value: "1000", Unit: types.UnitOfMeasureWh}
	suite.conn.meterUpdated = suite.clock.Now()
	suite.conn.txnId = 1

	// missing power during an active txn must still surface as not available
	_, err := suite.conn.CurrentPower()
	suite.Equal(api.ErrNotAvailable, err, "CurrentPower")
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

// TestOnStatusNotificationClearsStaleTxn ensures that a transaction left over
// from a previous session (e.g. because the charger never sent StopTransaction,
// like the Zaptec Go 2 in local OCPP mode) is cleared when the connector
// returns to Available, so the next Preparing can trigger RemoteStartTransaction.
func (suite *connTestSuite) TestOnStatusNotificationClearsStaleTxn() {
	suite.conn.remoteIdTag = "evcc"
	suite.conn.txnId = 42
	suite.conn.idTag = "stale"

	_, err := suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusAvailable,
		ErrorCode:   core.NoError,
		Timestamp:   types.NewDateTime(suite.clock.Now()),
	})
	suite.NoError(err)
	suite.Equal(0, suite.conn.txnId, "txnId should be cleared on Available")
	suite.Equal("", suite.conn.idTag, "idTag should be cleared on Available")

	// next Preparing notification must now satisfy NeedsAuthentication
	_, err = suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusPreparing,
		ErrorCode:   core.NoError,
		Timestamp:   types.NewDateTime(suite.clock.Now().Add(time.Second)),
	})
	suite.NoError(err)
	suite.True(suite.conn.NeedsAuthentication(), "Preparing after Available should require authentication")
}

// TestOnBootNotificationClearsStaleTxn ensures a transaction left over from
// before a charge point reboot is cleared on BootNotification, so a connector
// that reconnects straight into Preparing (never reporting Available, e.g.
// Grizzl-E) can still trigger RemoteStartTransaction.
func (suite *connTestSuite) TestOnBootNotificationClearsStaleTxn() {
	suite.conn.remoteIdTag = "evcc"
	suite.conn.txnId = 42
	suite.conn.idTag = "stale"

	// charge point reboots and re-announces itself
	_, err := suite.cp.OnBootNotification(&core.BootNotificationRequest{
		ChargePointModel:  "GRM 2024",
		ChargePointVendor: "UnitedChargers",
	})
	suite.NoError(err)
	suite.Equal(0, suite.conn.txnId, "txnId should be cleared on reboot")
	suite.Equal("", suite.conn.idTag, "idTag should be cleared on reboot")

	// connector reconnects straight into Preparing, without an Available in
	// between and without a timestamp (as the Grizzl-E does)
	_, err = suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusPreparing,
		ErrorCode:   core.NoError,
	})
	suite.NoError(err)
	suite.True(suite.conn.NeedsAuthentication(), "Preparing after reboot should require authentication")
}

// TestOnBootNotificationBlocksStaleMeterValueRecovery ensures MeterValues
// arriving between the BootNotification and the first post-reboot status cannot
// resurrect the transaction the reboot just cleared. The cached status still
// says Charging, which would otherwise qualify the transaction id for recovery
// and suppress RemoteStartTransaction all over again. Setup solicits MeterValues
// on every reboot and never a StatusNotification, so this is the normal
// post-reboot ordering rather than a rare race.
func (suite *connTestSuite) TestOnBootNotificationBlocksStaleMeterValueRecovery() {
	suite.conn.remoteIdTag = "evcc"

	// charging before the power cut
	_, err := suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusCharging,
		ErrorCode:   core.NoError,
		Timestamp:   types.NewDateTime(suite.clock.Now()),
	})
	suite.NoError(err)

	suite.conn.txnId = 42
	suite.conn.idTag = "stale"

	// charge point reboots and re-announces itself
	_, err = suite.cp.OnBootNotification(&core.BootNotificationRequest{
		ChargePointModel:  "GRM 2024",
		ChargePointVendor: "UnitedChargers",
	})
	suite.NoError(err)
	suite.Equal(0, suite.conn.txnId, "txnId should be cleared on reboot")

	// the stale transaction id must not be adopted from meter values
	txnId := 42
	_, err = suite.conn.OnMeterValues(&core.MeterValuesRequest{
		ConnectorId:   1,
		TransactionId: &txnId,
		MeterValue: []types.MeterValue{{
			Timestamp:    types.NewDateTime(suite.clock.Now()),
			SampledValue: []types.SampledValue{{Measurand: types.MeasurandCurrentImport, Value: "0"}},
		}},
	})
	suite.NoError(err)
	suite.Equal(0, suite.conn.txnId, "stale transaction must not be recovered before a fresh status")

	// connector reports Preparing without a timestamp (as the Grizzl-E does)
	_, err = suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusPreparing,
		ErrorCode:   core.NoError,
	})
	suite.NoError(err)
	suite.True(suite.conn.NeedsAuthentication(), "Preparing after reboot should require authentication")
}

// TestMeterValueRecoveryAfterFreshStatus ensures the reboot gate only defers
// recovery rather than disabling it: once the charge point has reported a fresh
// status, a transaction id carried by MeterValues is adopted again. This is the
// path charger/ocpp.go relies on after a reconnect (#15951).
func (suite *connTestSuite) TestMeterValueRecoveryAfterFreshStatus() {
	suite.conn.txnId = 42

	_, err := suite.cp.OnBootNotification(&core.BootNotificationRequest{
		ChargePointModel:  "GRM 2024",
		ChargePointVendor: "UnitedChargers",
	})
	suite.NoError(err)
	suite.Equal(0, suite.conn.txnId, "txnId should be cleared on reboot")

	// charge point reports that it resumed charging
	_, err = suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusCharging,
		ErrorCode:   core.NoError,
	})
	suite.NoError(err)

	txnId := 77
	_, err = suite.conn.OnMeterValues(&core.MeterValuesRequest{
		ConnectorId:   1,
		TransactionId: &txnId,
		MeterValue: []types.MeterValue{{
			Timestamp:    types.NewDateTime(suite.clock.Now()),
			SampledValue: []types.SampledValue{{Measurand: types.MeasurandCurrentImport, Value: "0"}},
		}},
	})
	suite.NoError(err)
	suite.Equal(77, suite.conn.txnId, "transaction must be recovered once a fresh status was applied")
}

// TestOnStatusNotificationKeepsActiveTxn ensures that an active transaction is
// not cleared by transient status notifications other than Available.
func (suite *connTestSuite) TestOnStatusNotificationKeepsActiveTxn() {
	suite.conn.txnId = 42
	suite.conn.idTag = "active"

	for _, status := range []core.ChargePointStatus{
		core.ChargePointStatusCharging,
		core.ChargePointStatusSuspendedEV,
		core.ChargePointStatusSuspendedEVSE,
		core.ChargePointStatusFinishing,
	} {
		_, err := suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
			ConnectorId: 1,
			Status:      status,
			ErrorCode:   core.NoError,
			Timestamp:   types.NewDateTime(suite.clock.Now()),
		})
		suite.NoError(err)
		suite.Equalf(42, suite.conn.txnId, "txnId must survive %s", status)
		suite.Equalf("active", suite.conn.idTag, "idTag must survive %s", status)
		suite.clock.Add(time.Second)
	}
}

// TestOnStatusNotificationKeepsTxnOnIgnoredAvailable ensures we do not clear
// transaction state when an Available notification is rejected due to an
// outdated timestamp (i.e. the cached status remains the current one).
func (suite *connTestSuite) TestOnStatusNotificationKeepsTxnOnIgnoredAvailable() {
	// prime with a recent Charging status
	_, err := suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusCharging,
		ErrorCode:   core.NoError,
		Timestamp:   types.NewDateTime(suite.clock.Now()),
	})
	suite.NoError(err)
	suite.conn.txnId = 42
	suite.conn.idTag = "active"

	// out-of-order Available with an older timestamp must be ignored
	// and must not clear the running transaction
	_, err = suite.conn.OnStatusNotification(&core.StatusNotificationRequest{
		ConnectorId: 1,
		Status:      core.ChargePointStatusAvailable,
		ErrorCode:   core.NoError,
		Timestamp:   types.NewDateTime(suite.clock.Now().Add(-time.Minute)),
	})
	suite.NoError(err)
	suite.Equal(42, suite.conn.txnId, "txnId must survive ignored Available")
	suite.Equal("active", suite.conn.idTag, "idTag must survive ignored Available")
}

// meterValues feeds the given samples as a single MeterValues message
func (suite *connTestSuite) meterValues(samples ...types.SampledValue) {
	_, err := suite.conn.OnMeterValues(&core.MeterValuesRequest{
		ConnectorId: 1,
		MeterValue: []types.MeterValue{{
			Timestamp:    types.NewDateTime(suite.clock.Now()),
			SampledValue: samples,
		}},
	})
	suite.NoError(err)
}

func energySample(value string, ctx types.ReadingContext) types.SampledValue {
	return types.SampledValue{
		Value:     value,
		Context:   ctx,
		Measurand: types.MeasurandEnergyActiveImportRegister,
		Unit:      types.UnitOfMeasureWh,
	}
}

// Some chargers (e.g. Growatt THOR) report the same measurand twice in a single
// MeterValues message: the live Sample.Periodic value next to a static
// Transaction.Begin snapshot that never changes. The live reading must win,
// regardless of the order the samples appear in.
func (suite *connTestSuite) TestOnMeterValuesBoundaryContextLast() {
	suite.meterValues(
		energySample("241010", types.ReadingContextSamplePeriodic),
		energySample("240970", types.ReadingContextTransactionBegin),
	)

	res, err := suite.conn.TotalEnergy()
	suite.NoError(err, "TotalEnergy")
	suite.Equal(241.010, res, "live reading must not be shadowed by the boundary snapshot")
}

func (suite *connTestSuite) TestOnMeterValuesBoundaryContextFirst() {
	suite.meterValues(
		energySample("240970", types.ReadingContextTransactionBegin),
		energySample("241010", types.ReadingContextSamplePeriodic),
	)

	res, err := suite.conn.TotalEnergy()
	suite.NoError(err, "TotalEnergy")
	suite.Equal(241.010, res, "live reading must not be shadowed by the boundary snapshot")
}

// chargers reporting the transaction's start value only as Transaction.Begin
// must keep it as there is no live reading to prefer over it
func (suite *connTestSuite) TestOnMeterValuesBoundaryContextOnly() {
	suite.meterValues(energySample("240970", types.ReadingContextTransactionBegin))

	res, err := suite.conn.TotalEnergy()
	suite.NoError(err, "TotalEnergy")
	suite.Equal(240.970, res, "boundary snapshot must be applied when no live reading is present")
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
