package ocpp

import (
	"testing"
	"time"

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

// While globally disabled, ApplyReportRules must never start a connection
// (no dial attempt at all - toStart stays empty), but must still keep the
// rules themselves so ReportRules()/status endpoints reflect real config and
// SetReportEnabled(true) has something to reconnect once flipped back on.
func TestApplyReportRulesSkipsConnectionsWhenGloballyDisabled(t *testing.T) {
	reportMu.Lock()
	connections = make(map[string]*reportConnection)
	reportEnabled = false
	reportMu.Unlock()
	t.Cleanup(func() {
		reportMu.Lock()
		reportEnabled = true
		reportMu.Unlock()
	})

	rule := ReportRule{LoadpointTitle: "Carport", UpstreamURL: "wss://a", StationID: "s1", IdTag: "EVCC"}
	ApplyReportRules([]ReportRule{rule})

	reportMu.RLock()
	defer reportMu.RUnlock()
	assert.Empty(t, connections, "no connection should be started while globally disabled")
	assert.Equal(t, []ReportRule{rule}, reportRules, "rules must survive being globally disabled")
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

func TestReportSessionLifecycleQueuesEvents(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	reportMu.Lock()
	connections = map[string]*reportConnection{"Carport": conn}
	reportMu.Unlock()

	ReportSessionStart("Carport", 1000)
	ReportMeterValue("Carport", 1200)
	ReportSessionStop("Carport", 1500)

	conn.mu.Lock()
	defer conn.mu.Unlock()
	require.Len(t, conn.events, 3)
	assert.Equal(t, reportEventStart, conn.events[0].kind)
	assert.Equal(t, 1000.0, conn.events[0].wh)
	assert.Equal(t, reportEventMeter, conn.events[1].kind)
	assert.Equal(t, 1200.0, conn.events[1].wh)
	assert.Equal(t, reportEventStop, conn.events[2].kind)
	assert.Equal(t, 1500.0, conn.events[2].wh)
	assert.False(t, conn.sessionActive)
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
	assert.Empty(t, conn.events)
}

// A stop for a loadpoint with no active reported session must not queue a
// stop - reconcile would otherwise try to stop a transaction that was never
// started.
func TestReportSessionStopNoOpWithoutActiveSession(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	reportMu.Lock()
	connections = map[string]*reportConnection{"Carport": conn}
	reportMu.Unlock()

	ReportSessionStop("Carport", 500)
	conn.mu.Lock()
	defer conn.mu.Unlock()
	assert.Empty(t, conn.events)
}

// reconcile must not attempt BootNotification/Authorize/StartTransaction
// against a disconnected connection, and queued events must survive the no-op
// so they are sent in order once a connection is established.
func TestReconcileNoOpWhenDisconnected(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	conn.mu.Lock()
	conn.sessionActive = true
	conn.push(reportEvent{kind: reportEventStart, at: time.Now(), wh: 1000})
	conn.mu.Unlock()

	require.False(t, conn.cp.IsConnected())
	assert.NotPanics(t, conn.reconcile)

	conn.mu.Lock()
	defer conn.mu.Unlock()
	assert.Len(t, conn.events, 1)
	assert.Nil(t, conn.transactionId)
	assert.False(t, conn.booted)
}

// A full queue drops the oldest meter sample first, so session boundaries are kept
func TestReportQueueDropsMeterSamplesFirst(t *testing.T) {
	conn := newUnstartedConnection("Carport")
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.push(reportEvent{kind: reportEventStart, at: time.Now(), wh: 1000})
	for i := 0; i < maxQueuedEvents; i++ {
		conn.push(reportEvent{kind: reportEventMeter, at: time.Now(), wh: float64(i)})
	}
	conn.push(reportEvent{kind: reportEventStop, at: time.Now(), wh: 2000})

	require.Len(t, conn.events, maxQueuedEvents)
	assert.Equal(t, reportEventStart, conn.events[0].kind)
	assert.Equal(t, reportEventStop, conn.events[len(conn.events)-1].kind)
}
