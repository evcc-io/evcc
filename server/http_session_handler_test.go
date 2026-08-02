package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestSessionHandlerTimezoneFilter(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(new(session.Session)))

	// Build in the process zone SQLite's 'localtime' uses (baked at startup).
	// Just past local midnight May 1 is still April 30 UTC east of UTC.
	ts := time.Date(2026, 5, 1, 0, 1, 0, 0, time.Local)
	require.NoError(t, db.Instance.Create(&session.Session{
		Created:       ts,
		Finished:      ts.Add(time.Hour),
		ChargedEnergy: 1.0,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/?year=2026&month=5", nil)
	rec := httptest.NewRecorder()
	sessionHandler(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var got session.Sessions
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
}
