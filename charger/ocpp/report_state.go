package ocpp

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
)

// persistedReportState is what survives a restart: the open transaction, the last
// register value reported for it, and the events not yet confirmed upstream
type persistedReportState struct {
	TransactionId *int             `json:"transactionId,omitempty"`
	LastWh        float64          `json:"lastWh"`
	LastAt        time.Time        `json:"lastAt"`
	Events        []persistedEvent `json:"events,omitempty"`
}

type persistedEvent struct {
	Kind reportEventKind `json:"kind"`
	At   time.Time       `json:"at"`
	Wh   float64         `json:"wh"`
}

// reportStateStore persists the state of each report connection
type reportStateStore interface {
	load(loadpointId int) (persistedReportState, bool)
	save(loadpointId int, st persistedReportState)
}

// reportStore is the store in use; tests replace it
var reportStore reportStateStore = reportSettingsStore{}

type reportSettingsStore struct{}

func reportStateKey(loadpointId int) string {
	return fmt.Sprintf("%s%d", keys.OcppReportState, loadpointId)
}

func (reportSettingsStore) load(loadpointId int) (persistedReportState, bool) {
	var st persistedReportState
	if err := settings.Json(reportStateKey(loadpointId), &st); err != nil {
		return st, false
	}
	return st, true
}

func (reportSettingsStore) save(loadpointId int, st persistedReportState) {
	if err := settings.SetJson(reportStateKey(loadpointId), st); err != nil {
		reportLog.ERROR.Printf("save report state %d: %v", loadpointId, err)
	}
}

// persistLocked saves the connection's state. Caller holds conn.mu.
func (conn *reportConnection) persistLocked() {
	st := persistedReportState{
		TransactionId: conn.transactionId,
		LastWh:        conn.lastWh,
		LastAt:        conn.lastAt,
	}
	for _, ev := range conn.events {
		st.Events = append(st.Events, persistedEvent{Kind: ev.kind, At: ev.at, Wh: ev.wh})
	}
	reportStore.save(conn.rule.LoadpointId, st)
}

// restoreLocked reloads the state saved before a restart. A transaction that was
// still open is ended at its last reported value, since the session it belonged
// to ended with the process. Caller holds conn.mu.
func (conn *reportConnection) restoreLocked() {
	st, ok := reportStore.load(conn.rule.LoadpointId)
	if !ok {
		return
	}

	conn.transactionId = st.TransactionId
	conn.lastWh = st.LastWh
	conn.lastAt = st.LastAt

	for _, ev := range st.Events {
		// a start that was already confirmed must not be sent again
		if ev.Kind == reportEventStart && conn.transactionId != nil {
			continue
		}
		conn.events = append(conn.events, reportEvent{kind: ev.Kind, at: ev.At, wh: ev.Wh})
	}

	if conn.transactionId != nil && !queuesStop(conn.events) {
		conn.events = append(conn.events, reportEvent{kind: reportEventStop, at: st.LastAt, wh: st.LastWh})
	}
}

func queuesStop(events []reportEvent) bool {
	for _, ev := range events {
		if ev.kind == reportEventStop {
			return true
		}
	}
	return false
}
