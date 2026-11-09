package ocpp

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type profileStartCentralSystem struct {
	ocpp16.CentralSystem
	calls chan struct{}
	err   error
}

func (cs *profileStartCentralSystem) RemoteStartTransaction(_ string, reply func(*core.RemoteStartTransactionConfirmation, error), _ string, _ ...func(*core.RemoteStartTransactionRequest)) error {
	cs.calls <- struct{}{}
	reply(core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusAccepted), cs.err)
	return nil
}

func (suite *connTestSuite) TestProfileStartInvalidatedBeforeEnqueue() {
	for _, event := range []string{"local start", "reconnect"} {
		suite.Run(event, func() {
			t := suite.T()
			conn := suite.conn
			cs := &profileStartCentralSystem{calls: make(chan struct{}, 1)}
			conn.cp.cs = &CS{CentralSystem: cs}
			conn.remoteIdTag = "test"
			conn.txnId = 0
			conn.status = &core.StatusNotificationRequest{Status: core.ChargePointStatusPreparing, ErrorCode: core.NoError}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			p := &profileRemoteStart{ctx: t.Context(), cancel: cancel, armed: true, limit: 4830}
			conn.profileStart = p

			// Block the dispatch at the charge point lock and deliver the event before enqueue.
			conn.cp.mu.Lock()
			locked := true
			defer func() {
				if locked {
					conn.cp.mu.Unlock()
				}
			}()
			result := make(chan bool, 1)
			go func() {
				attempted, _ := conn.dispatchProfileStart(ctx, 0, 4830, 4830)
				result <- attempted
			}()
			suite.Require().Eventually(func() bool {
				if p.requests.TryLock() {
					p.requests.Unlock()
					return false
				}
				return true
			}, time.Second, time.Millisecond)

			if event == "local start" {
				conn.startTransaction(42, "local")
			} else {
				conn.cp.setConnected(false)
				conn.cp.setConnected(true)
			}
			conn.cp.mu.Unlock()
			locked = false

			select {
			case attempted := <-result:
				suite.False(attempted)
				suite.Empty(cs.calls)
			case <-time.After(time.Second):
				t.Fatal("dispatch did not finish")
			}
		})
	}
}

func (suite *connTestSuite) TestProfileStartTimeoutDoesNotRetry() {
	conn := suite.conn
	cs := &profileStartCentralSystem{calls: make(chan struct{}, 1), err: api.ErrTimeout}
	conn.cp.cs = &CS{CentralSystem: cs}
	conn.remoteIdTag = "test"
	conn.status = &core.StatusNotificationRequest{Status: core.ChargePointStatusPreparing, ErrorCode: core.NoError}
	conn.profileStart = &profileRemoteStart{ctx: suite.T().Context(), armed: true, limit: 4830}

	attempted, retry := conn.dispatchProfileStart(suite.T().Context(), 0, 4830, 4830)
	suite.True(attempted)
	suite.False(retry)
	suite.False(conn.profileStart.armed)
	suite.Len(cs.calls, 1)
}

func (suite *connTestSuite) TestProfileStartIgnoresReplayedMeterValues() {
	conn := suite.conn
	conn.profileStart = &profileRemoteStart{ctx: suite.T().Context(), armed: true, paused: true}
	conn.status = &core.StatusNotificationRequest{Status: core.ChargePointStatusCharging, ErrorCode: core.NoError}
	conn.txnId = 42
	suite.clock.Add(time.Minute)
	now := suite.clock.Now()
	_, err := conn.OnStopTransaction(&core.StopTransactionRequest{TransactionId: 42})
	suite.Require().NoError(err)

	for _, tc := range []struct {
		id      int
		time    time.Time
		context types.ReadingContext
	}{
		{id: 42, time: now, context: types.ReadingContextSamplePeriodic},
		{id: 43, time: now.Add(-10 * time.Second), context: types.ReadingContextSamplePeriodic},
		{id: 43, time: now, context: types.ReadingContextTransactionEnd},
	} {
		_, err := conn.OnMeterValues(&core.MeterValuesRequest{
			TransactionId: &tc.id,
			MeterValue: []types.MeterValue{{
				Timestamp: types.NewDateTime(tc.time),
				SampledValue: []types.SampledValue{{
					Context: tc.context, Measurand: types.MeasurandPowerActiveImport, Value: "4600",
				}},
			}},
		})
		suite.Require().NoError(err)
		suite.Zero(conn.txnId)
	}

	_, err = conn.OnStatusNotification(&core.StatusNotificationRequest{Status: core.ChargePointStatusPreparing, ErrorCode: core.NoError})
	suite.Require().NoError(err)
	suite.True(conn.NeedsAuthentication())
}

func (suite *connTestSuite) TestProfileStartReconcilesTransactionAfterReconnect() {
	conn := suite.conn
	for _, fresh := range []bool{false, true} {
		conn.profileStart = &profileRemoteStart{ctx: suite.T().Context()}
		conn.status = &core.StatusNotificationRequest{Status: core.ChargePointStatusCharging, ErrorCode: core.NoError}
		conn.txnId = 42
		conn.cp.connect(false)
		suite.clock.Add(time.Second)
		conn.cp.connect(true)
		_, err := conn.OnStatusNotification(&core.StatusNotificationRequest{Status: core.ChargePointStatusCharging, ErrorCode: core.NoError})
		suite.Require().NoError(err)

		stamp := suite.clock.Now()
		if !fresh {
			stamp = stamp.Add(-2 * time.Second)
		}
		_, err = conn.OnMeterValues(&core.MeterValuesRequest{
			TransactionId: new(42),
			MeterValue: []types.MeterValue{{
				Timestamp: types.NewDateTime(stamp),
				SampledValue: []types.SampledValue{{
					Measurand: types.MeasurandPowerActiveImport, Value: "4600",
				}},
			}},
		})
		suite.Require().NoError(err)
		_, err = conn.OnStatusNotification(&core.StatusNotificationRequest{Status: core.ChargePointStatusPreparing, ErrorCode: core.NoError})
		suite.Require().NoError(err)
		suite.Equal(!fresh, conn.NeedsAuthentication(), "fresh meter data confirms the retained transaction")
	}
}
