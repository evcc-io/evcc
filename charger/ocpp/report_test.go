package ocpp

import (
	"testing"

	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	occore "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUnstartedConnection builds a reportConnection with a real but never-
// dialed ocpp16.ChargePoint - construction alone does no network I/O, so
// cp.IsConnected() is reliably false, letting reconcile()'s disconnected
// early-return be exercised without a live upstream.
func newUnstartedConnection(title string) *reportConnection {
	client := ws.NewClient()
	endpoint := ocppj.NewClient(title, client, nil, nil, occore.Profile, remotetrigger.Profile)
	return &reportConnection{
		title: title,
		cp:    ocpp16.NewChargePoint(title, endpoint, client),
		jobs:  make(chan func(), 16),
	}
}

func TestReportRuleSameConnection(t *testing.T) {
	base := ReportRule{LoadpointTitle: "Carport", UpstreamURL: "wss://a", StationID: "s1", Username: "u", Password: "p"}

	same := base
	same.IdTag = "changed" // idTag doesn't require a reconnect
	assert.True(t, base.sameConnection(same))

	diff := base
	diff.UpstreamURL = "wss://b"
	assert.False(t, base.sameConnection(diff))
}

func TestReportRuleRedacted(t *testing.T) {
	r := ReportRule{Password: "secret", CaCert: "cert-data"}
	red := r.Redacted()
	assert.NotEqual(t, "secret", red.Password)
	assert.NotEqual(t, "cert-data", red.CaCert)
	assert.NotEmpty(t, red.Password)
}

// The report client is a one-way reporter (evcc-io/evcc#32989): its upstream
// is a billing/certification backend, not an access-control authority, so
// remote start/stop is always honestly rejected regardless of connector,
// transaction id, or whether a loadpoint is even configured for this rule.
func TestOnRemoteStartTransactionAlwaysRejected(t *testing.T) {
	handler := &reportHandler{conn: &reportConnection{title: "Carport"}}

	res, err := handler.OnRemoteStartTransaction(&occore.RemoteStartTransactionRequest{IdTag: "tag1"})
	require.NoError(t, err)
	assert.Equal(t, types.RemoteStartStopStatusRejected, res.Status)
}

func TestOnRemoteStopTransactionAlwaysRejected(t *testing.T) {
	txID := 42
	handler := &reportHandler{conn: &reportConnection{title: "Carport", transactionId: &txID}}

	res, err := handler.OnRemoteStopTransaction(&occore.RemoteStopTransactionRequest{TransactionId: 42})
	require.NoError(t, err)
	assert.Equal(t, types.RemoteStartStopStatusRejected, res.Status)
}

func TestOnUnlockConnectorNotSupported(t *testing.T) {
	handler := &reportHandler{conn: &reportConnection{title: "Carport"}}

	res, err := handler.OnUnlockConnector(&occore.UnlockConnectorRequest{ConnectorId: 1})
	require.NoError(t, err)
	assert.Equal(t, occore.UnlockStatusNotSupported, res.Status)
}

func TestApplyReportRulesNoOpForUnconfiguredLoadpoint(t *testing.T) {
	// ReportSessionStart/MeterValue/Stop must be safe no-ops when no rule
	// is configured for the given loadpoint - this is the common case.
	reportMu.Lock()
	connections = make(map[string]*reportConnection)
	reportMu.Unlock()

	assert.NotPanics(t, func() {
		ReportSessionStart("unknown", 1000)
		ReportMeterValue("unknown", 500)
		ReportSessionStop("unknown", 1500)
	})
}

// ReportSessionStart/MeterValue/Stop must record durable desired state before
// ever touching the network, so a call made while offline (or one whose
// reconcile job gets dropped under backpressure) is retried by the next
// reconcile rather than silently lost - see reportConnection.reconcile.
func TestReportSessionLifecycleRecordsDurableState(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	reportMu.Lock()
	connections = map[string]*reportConnection{"Carport": conn}
	reportMu.Unlock()

	ReportSessionStart("Carport", 1000)
	conn.mu.Lock()
	assert.True(t, conn.sessionActive)
	assert.Equal(t, 1000.0, conn.meterStartWh)
	assert.Equal(t, 1000.0, conn.lastMeterWh)
	assert.False(t, conn.pendingStop)
	conn.mu.Unlock()

	ReportMeterValue("Carport", 1200)
	conn.mu.Lock()
	assert.Equal(t, 1200.0, conn.lastMeterWh)
	conn.mu.Unlock()

	ReportSessionStop("Carport", 1500)
	conn.mu.Lock()
	assert.True(t, conn.pendingStop)
	assert.Equal(t, 1500.0, conn.meterStopWh)
	// sessionActive is only cleared once reconcile confirms the
	// StopTransaction, not at the moment the stop is requested
	assert.True(t, conn.sessionActive)
	conn.mu.Unlock()
}

// A meter update for a loadpoint with no active reported session (no
// ReportSessionStart yet, or already stopped) must not fabricate session
// state - there's nothing to attach the reading to.
func TestReportMeterValueNoOpWithoutActiveSession(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	reportMu.Lock()
	connections = map[string]*reportConnection{"Carport": conn}
	reportMu.Unlock()

	ReportMeterValue("Carport", 999)
	conn.mu.Lock()
	defer conn.mu.Unlock()
	assert.False(t, conn.sessionActive)
	assert.Zero(t, conn.lastMeterWh)
}

// A stop for a loadpoint with no active reported session must not set
// pendingStop - reconcile would otherwise try to stop a transaction that was
// never started.
func TestReportSessionStopNoOpWithoutActiveSession(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	reportMu.Lock()
	connections = map[string]*reportConnection{"Carport": conn}
	reportMu.Unlock()

	ReportSessionStop("Carport", 500)
	conn.mu.Lock()
	defer conn.mu.Unlock()
	assert.False(t, conn.pendingStop)
}

// reconcile must not attempt BootNotification/Authorize/StartTransaction
// against a disconnected connection - an attempt against a dead connection
// is certain to fail and, unlike the durable state above, would previously
// have been the only attempt made (see the original bug this replaced).
func TestReconcileNoOpWhenDisconnected(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	conn.sessionActive = true
	conn.meterStartWh = 1000
	conn.lastMeterWh = 1000

	require.False(t, conn.cp.IsConnected())
	assert.NotPanics(t, conn.reconcile)

	conn.mu.Lock()
	defer conn.mu.Unlock()
	// desired state must survive the no-op reconcile untouched, ready for
	// the next reconcile once a connection is established
	assert.True(t, conn.sessionActive)
	assert.Nil(t, conn.transactionId)
	assert.False(t, conn.booted)
}
