package ocpp

import (
	"testing"

	occore "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
