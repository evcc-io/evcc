package ocpp

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	occore "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func withLookup(t *testing.T, title string, lp loadpoint.API) {
	t.Helper()
	SetLoadpointLookup(func(s string) (loadpoint.API, bool) {
		if s == title {
			return lp, true
		}
		return nil, false
	})
	t.Cleanup(func() { SetLoadpointLookup(nil) })
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

func TestOnRemoteStartTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	lp := loadpoint.NewMockAPI(ctrl)

	conn := &reportConnection{title: "Carport"}
	handler := &reportHandler{conn: conn}

	t.Run("accepted, connector 1", func(t *testing.T) {
		withLookup(t, "Carport", lp)
		lp.EXPECT().SetMode(api.ModeNow)

		res, err := handler.OnRemoteStartTransaction(&occore.RemoteStartTransactionRequest{IdTag: "tag1"})
		require.NoError(t, err)
		assert.Equal(t, types.RemoteStartStopStatusAccepted, res.Status)
	})

	t.Run("rejected, wrong connector", func(t *testing.T) {
		withLookup(t, "Carport", lp)
		two := 2

		res, err := handler.OnRemoteStartTransaction(&occore.RemoteStartTransactionRequest{IdTag: "tag1", ConnectorId: &two})
		require.NoError(t, err)
		assert.Equal(t, types.RemoteStartStopStatusRejected, res.Status)
	})

	t.Run("rejected, no such loadpoint", func(t *testing.T) {
		SetLoadpointLookup(func(string) (loadpoint.API, bool) { return nil, false })
		t.Cleanup(func() { SetLoadpointLookup(nil) })

		res, err := handler.OnRemoteStartTransaction(&occore.RemoteStartTransactionRequest{IdTag: "tag1"})
		require.NoError(t, err)
		assert.Equal(t, types.RemoteStartStopStatusRejected, res.Status)
	})
}

func TestOnRemoteStopTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	lp := loadpoint.NewMockAPI(ctrl)

	txID := 42
	conn := &reportConnection{title: "Carport", transactionId: &txID}
	handler := &reportHandler{conn: conn}

	t.Run("accepted, matching transaction", func(t *testing.T) {
		withLookup(t, "Carport", lp)
		lp.EXPECT().SetMode(api.ModeOff)

		res, err := handler.OnRemoteStopTransaction(&occore.RemoteStopTransactionRequest{TransactionId: 42})
		require.NoError(t, err)
		assert.Equal(t, types.RemoteStartStopStatusAccepted, res.Status)
	})

	t.Run("rejected, mismatched transaction", func(t *testing.T) {
		withLookup(t, "Carport", lp)

		res, err := handler.OnRemoteStopTransaction(&occore.RemoteStopTransactionRequest{TransactionId: 999})
		require.NoError(t, err)
		assert.Equal(t, types.RemoteStartStopStatusRejected, res.Status)
	})

	t.Run("rejected, no active transaction", func(t *testing.T) {
		withLookup(t, "Carport", lp)
		empty := &reportConnection{title: "Carport"}
		emptyHandler := &reportHandler{conn: empty}

		res, err := emptyHandler.OnRemoteStopTransaction(&occore.RemoteStopTransactionRequest{TransactionId: 42})
		require.NoError(t, err)
		assert.Equal(t, types.RemoteStartStopStatusRejected, res.Status)
	})
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
