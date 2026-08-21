package ocpp

// Bidirectional OCPP client, one connection per loadpoint. Unlike the forwarder
// (which relays real OCPP frames between a charger that already speaks OCPP to
// evcc and an upstream Central System), this dials OUT to an upstream Central
// System and SYNTHESIZES OCPP messages from evcc's own charging session
// lifecycle - so it works for any charger type (Modbus etc.), not just
// OCPP-native ones. It also accepts upstream remote-control commands
// (RemoteStartTransaction/RemoteStopTransaction) and translates them into
// evcc's own loadpoint control.
//
// Design: evcc-io/evcc#32989.

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

var reportLog = util.NewLogger("ocpp-report")

// ReportRule configures reporting a loadpoint's charging sessions to an
// upstream OCPP 1.6J Central System, and accepting remote control from it.
type ReportRule struct {
	LoadpointTitle string `json:"loadpointTitle" yaml:"loadpointTitle"`
	UpstreamURL    string `json:"upstreamUrl" yaml:"upstreamUrl"`
	StationID      string `json:"stationId" yaml:"stationId"`
	IdTag          string `json:"idTag,omitempty" yaml:"idTag,omitempty"`
	Username       string `json:"username,omitempty" yaml:"username,omitempty"`
	Password       string `json:"password,omitempty" yaml:"password,omitempty"`
	Insecure       bool   `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	CaCert         string `json:"caCert,omitempty" yaml:"caCert,omitempty"`
}

func (r ReportRule) Redacted() ReportRule {
	r.Password = util.Masked(r.Password)
	r.CaCert = util.Masked(r.CaCert)
	return r
}

// sameConnection reports whether two rules dial upstream identically (no
// reconnect needed).
func (r ReportRule) sameConnection(o ReportRule) bool {
	return r.UpstreamURL == o.UpstreamURL && r.StationID == o.StationID &&
		r.Username == o.Username && r.Password == o.Password &&
		r.Insecure == o.Insecure && r.CaCert == o.CaCert
}

// ReportSessionStatus is a snapshot of one report connection for the UI.
type ReportSessionStatus struct {
	LoadpointTitle    string `json:"loadpointTitle"`
	UpstreamURL       string `json:"upstreamUrl"`
	UpstreamConnected bool   `json:"upstreamConnected"`
	Error             string `json:"error,omitempty"`
}

var (
	reportMu     sync.RWMutex
	reportRules  []ReportRule
	connections  = make(map[string]*reportConnection) // keyed by LoadpointTitle
	reportErrors = make(map[string]string)

	reportCbMu      sync.Mutex
	reportUpdatedCb func()

	loadpointLookupMu sync.RWMutex
	loadpointLookupFn func(title string) (loadpoint.API, bool)
)

// SetLoadpointLookup registers the function used to resolve a loadpoint by
// title, for both outbound reporting and inbound remote-control dispatch.
// Called once at site startup.
func SetLoadpointLookup(fn func(title string) (loadpoint.API, bool)) {
	loadpointLookupMu.Lock()
	loadpointLookupFn = fn
	loadpointLookupMu.Unlock()
}

func lookupLoadpoint(title string) (loadpoint.API, bool) {
	loadpointLookupMu.RLock()
	fn := loadpointLookupFn
	loadpointLookupMu.RUnlock()
	if fn == nil {
		return nil, false
	}
	return fn(title)
}

// SetReportUpdated registers a callback fired when a connection's status changes.
func SetReportUpdated(cb func()) {
	reportCbMu.Lock()
	reportUpdatedCb = cb
	reportCbMu.Unlock()
}

func notifyReportUpdated() {
	reportCbMu.Lock()
	cb := reportUpdatedCb
	reportCbMu.Unlock()
	if cb != nil {
		cb()
	}
}

// ReportEnabled returns true when at least one report rule is configured.
func ReportEnabled() bool {
	reportMu.RLock()
	defer reportMu.RUnlock()
	return len(reportRules) > 0
}

// ReportRules returns the currently configured report rules.
func ReportRules() []ReportRule {
	reportMu.RLock()
	defer reportMu.RUnlock()
	return slices.Clone(reportRules)
}

// GetReportStatus returns a snapshot of all configured report connections.
func GetReportStatus() []ReportSessionStatus {
	reportMu.RLock()
	defer reportMu.RUnlock()

	out := make([]ReportSessionStatus, 0, len(reportRules))
	for _, r := range reportRules {
		st := ReportSessionStatus{
			LoadpointTitle: r.LoadpointTitle,
			UpstreamURL:    strings.TrimRight(r.UpstreamURL, "/"),
		}
		if conn, ok := connections[r.LoadpointTitle]; ok && conn.cp.IsConnected() {
			st.UpstreamConnected = true
		} else if msg, ok := reportErrors[r.LoadpointTitle]; ok {
			st.Error = msg
		}
		out = append(out, st)
	}
	return out
}

func recordReportError(title, msg string) {
	reportMu.Lock()
	reportErrors[title] = msg
	reportMu.Unlock()
	notifyReportUpdated()
}

func clearReportError(title string) {
	reportMu.Lock()
	_, had := reportErrors[title]
	delete(reportErrors, title)
	reportMu.Unlock()
	if had {
		notifyReportUpdated()
	}
}

// ApplyReportRules replaces the report rules and (re)dials connections.
func ApplyReportRules(rules []ReportRule) {
	reportMu.Lock()
	reportRules = rules

	valid := make(map[string]bool, len(rules))
	for _, r := range rules {
		valid[r.LoadpointTitle] = true
	}

	var stale []*reportConnection
	for title, conn := range connections {
		if !valid[title] {
			stale = append(stale, conn)
			delete(connections, title)
			delete(reportErrors, title)
		}
	}

	var toStart []*reportConnection
	for _, r := range rules {
		if old, ok := connections[r.LoadpointTitle]; ok {
			if old.rule.sameConnection(r) {
				old.rule = r // idTag etc. may have changed, doesn't need a reconnect
				continue
			}
			stale = append(stale, old)
		}
		conn := newReportConnection(r)
		connections[r.LoadpointTitle] = conn
		toStart = append(toStart, conn)
	}
	reportMu.Unlock()

	for _, conn := range stale {
		conn.close()
	}
	for _, conn := range toStart {
		go conn.run()
	}

	notifyReportUpdated()
}

// reportConnection is one loadpoint's OCPP client connection to its upstream.
type reportConnection struct {
	title string
	rule  ReportRule
	cp    ocpp16.ChargePoint
	jobs  chan func()
	done  chan struct{}

	mu            sync.Mutex
	booted        bool
	transactionId *int
}

func newReportConnection(rule ReportRule) *reportConnection {
	conn := &reportConnection{
		title: rule.LoadpointTitle,
		rule:  rule,
		jobs:  make(chan func(), 16),
		done:  make(chan struct{}),
	}

	var opts []ws.ClientOpt
	if rule.Insecure || rule.CaCert != "" {
		tlsConfig := &tls.Config{InsecureSkipVerify: rule.Insecure}
		if rule.CaCert != "" {
			pool := x509.NewCertPool()
			if pool.AppendCertsFromPEM([]byte(rule.CaCert)) {
				tlsConfig.RootCAs = pool
			}
		}
		opts = append(opts, ws.WithClientTLSConfig(tlsConfig))
	}

	client := ws.NewClient(opts...)
	client.SetRequestedSubProtocol(types.V16Subprotocol)
	if rule.Username != "" || rule.Password != "" {
		client.SetBasicAuth(rule.Username, rule.Password)
	}

	stationID := rule.StationID
	if stationID == "" {
		stationID = "evcc-" + rule.LoadpointTitle
	}

	endpoint := ocppj.NewClient(stationID, client, nil, nil, core.Profile, remotetrigger.Profile)
	handler := &reportHandler{conn: conn}

	cp := ocpp16.NewChargePoint(stationID, endpoint, client)
	cp.SetCoreHandler(handler)
	cp.SetRemoteTriggerHandler(handler)
	conn.cp = cp

	endpoint.SetOnDisconnectedHandler(func(err error) {
		msg := "disconnected"
		if err != nil {
			msg = err.Error()
		}
		recordReportError(conn.title, msg)
	})
	endpoint.SetOnReconnectedHandler(func() {
		clearReportError(conn.title)
		conn.mu.Lock()
		conn.booted = false // re-boot after reconnect
		conn.mu.Unlock()
	})

	go func() {
		for err := range cp.Errors() {
			reportLog.DEBUG.Printf("%s: %v", conn.title, err)
		}
	}()

	return conn
}

// run dials the connection (with backoff) and drains its job queue. Runs
// until close() is called.
func (conn *reportConnection) run() {
	go conn.dial()

	for {
		select {
		case job := <-conn.jobs:
			job()
		case <-conn.done:
			return
		}
	}
}

func (conn *reportConnection) dial() {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // retry forever
	bo.MaxInterval = 5 * time.Minute

	op := func() error {
		select {
		case <-conn.done:
			return backoff.Permanent(errors.New("closed"))
		default:
		}
		if err := conn.cp.Start(conn.rule.UpstreamURL); err != nil {
			recordReportError(conn.title, err.Error())
			return err
		}
		clearReportError(conn.title)
		return nil
	}

	_ = backoff.Retry(op, backoff.WithMaxRetries(bo, 0))
}

func (conn *reportConnection) close() {
	close(conn.done)
	conn.cp.Stop()
}

// enqueue schedules a job on the connection's single worker goroutine, so
// StartTransaction/MeterValues/StopTransaction for one session stay ordered.
// Silently dropped if the queue is full (backpressure - a report connection
// that's badly behind shouldn't block the loadpoint's own control loop).
func (conn *reportConnection) enqueue(job func()) {
	select {
	case conn.jobs <- job:
	default:
		reportLog.WARN.Printf("%s: report queue full, dropping message", conn.title)
	}
}

func (conn *reportConnection) ensureBoot() {
	conn.mu.Lock()
	booted := conn.booted
	conn.mu.Unlock()
	if booted {
		return
	}
	if _, err := conn.cp.BootNotification("evcc", "evcc.io"); err != nil {
		reportLog.DEBUG.Printf("%s: boot notification: %v", conn.title, err)
		return
	}
	conn.mu.Lock()
	conn.booted = true
	conn.mu.Unlock()
}

func (conn *reportConnection) idTag() string {
	if conn.rule.IdTag != "" {
		return conn.rule.IdTag
	}
	return "EVCC"
}

// ReportSessionStart notifies the loadpoint's report connection (if any) that
// a charging session started. No-op if the loadpoint has no rule configured.
func ReportSessionStart(loadpointTitle string, meterStartWh float64) {
	reportMu.RLock()
	conn := connections[loadpointTitle]
	reportMu.RUnlock()
	if conn == nil {
		return
	}

	idTag := conn.idTag()
	conn.enqueue(func() {
		conn.ensureBoot()

		if _, err := conn.cp.Authorize(idTag); err != nil {
			reportLog.DEBUG.Printf("%s: authorize: %v", conn.title, err)
		}

		res, err := conn.cp.StartTransaction(1, idTag, int(meterStartWh), types.NewDateTime(time.Now()))
		if err != nil {
			reportLog.DEBUG.Printf("%s: start transaction: %v", conn.title, err)
			return
		}

		conn.mu.Lock()
		conn.transactionId = &res.TransactionId
		conn.mu.Unlock()
	})
}

// ReportMeterValue notifies the loadpoint's report connection (if any) of the
// current cumulative session energy. No-op without an active reported session.
func ReportMeterValue(loadpointTitle string, energyWh float64) {
	reportMu.RLock()
	conn := connections[loadpointTitle]
	reportMu.RUnlock()
	if conn == nil {
		return
	}

	conn.enqueue(func() {
		conn.mu.Lock()
		txID := conn.transactionId
		conn.mu.Unlock()
		if txID == nil {
			return
		}

		mv := types.MeterValue{
			Timestamp: types.NewDateTime(time.Now()),
			SampledValue: []types.SampledValue{{
				Value:     fmt.Sprintf("%.0f", energyWh),
				Measurand: types.MeasurandEnergyActiveImportRegister,
				Unit:      types.UnitOfMeasureWh,
			}},
		}
		if _, err := conn.cp.MeterValues(1, []types.MeterValue{mv}, func(r *core.MeterValuesRequest) {
			r.TransactionId = txID
		}); err != nil {
			reportLog.DEBUG.Printf("%s: meter values: %v", conn.title, err)
		}
	})
}

// ReportSessionStop notifies the loadpoint's report connection (if any) that
// the charging session ended.
func ReportSessionStop(loadpointTitle string, meterStopWh float64) {
	reportMu.RLock()
	conn := connections[loadpointTitle]
	reportMu.RUnlock()
	if conn == nil {
		return
	}

	conn.enqueue(func() {
		conn.mu.Lock()
		txID := conn.transactionId
		conn.transactionId = nil
		conn.mu.Unlock()
		if txID == nil {
			return
		}

		if _, err := conn.cp.StopTransaction(int(meterStopWh), types.NewDateTime(time.Now()), *txID); err != nil {
			reportLog.DEBUG.Printf("%s: stop transaction: %v", conn.title, err)
		}
	})
}

// reportHandler implements the inbound OCPP call handlers (remote control)
// for one report connection.
type reportHandler struct {
	conn *reportConnection
}

var _ core.ChargePointHandler = (*reportHandler)(nil)
var _ remotetrigger.ChargePointHandler = (*reportHandler)(nil)

func (h *reportHandler) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	if request.ConnectorId != nil && *request.ConnectorId != 1 {
		return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
	}

	lp, ok := lookupLoadpoint(h.conn.title)
	if !ok {
		return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
	}

	reportLog.DEBUG.Printf("%s: remote start requested", h.conn.title)
	lp.SetMode(api.ModeNow)

	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

func (h *reportHandler) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	h.conn.mu.Lock()
	txID := h.conn.transactionId
	h.conn.mu.Unlock()

	if txID == nil || *txID != request.TransactionId {
		return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
	}

	lp, ok := lookupLoadpoint(h.conn.title)
	if !ok {
		return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
	}

	reportLog.DEBUG.Printf("%s: remote stop requested", h.conn.title)
	lp.SetMode(api.ModeOff)

	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

// OnUnlockConnector: evcc has no generic cross-charger unlock capability, so
// this is an honest rejection rather than a fake success.
func (h *reportHandler) OnUnlockConnector(request *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusNotSupported), nil
}

// OnReset: no generic per-charger reset exists, and resetting evcc itself
// would be wrong. Benign no-op accept so upstream doesn't flag the station
// as faulty.
func (h *reportHandler) OnReset(request *core.ResetRequest) (*core.ResetConfirmation, error) {
	reportLog.DEBUG.Printf("%s: reset requested (no-op)", h.conn.title)
	return core.NewResetConfirmation(core.ResetStatusAccepted), nil
}

func (h *reportHandler) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusAccepted), nil
}

func (h *reportHandler) OnChangeConfiguration(request *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusAccepted), nil
}

func (h *reportHandler) OnClearCache(request *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	return core.NewClearCacheConfirmation(core.ClearCacheStatusAccepted), nil
}

func (h *reportHandler) OnDataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	return core.NewDataTransferConfirmation(core.DataTransferStatusUnknownVendorId), nil
}

func (h *reportHandler) OnGetConfiguration(request *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	return core.NewGetConfigurationConfirmation(nil), nil
}

// OnTriggerMessage re-emits the requested message from currently cached
// state, best-effort.
func (h *reportHandler) OnTriggerMessage(request *remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error) {
	switch request.RequestedMessage {
	case core.BootNotificationFeatureName:
		h.conn.mu.Lock()
		h.conn.booted = false
		h.conn.mu.Unlock()
		h.conn.enqueue(h.conn.ensureBoot)
	case core.MeterValuesFeatureName:
		if lp, ok := lookupLoadpoint(h.conn.title); ok {
			ReportMeterValue(h.conn.title, lp.GetChargedEnergy()*1e3)
		}
	}

	return remotetrigger.NewTriggerMessageConfirmation(remotetrigger.TriggerMessageStatusAccepted), nil
}
