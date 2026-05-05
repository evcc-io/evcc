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
	t.Setenv("TZ", "Europe/Berlin")
	loc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(new(session.Session)))

	// 2026-05-01 00:01 CEST = 2026-04-30 22:01 UTC: local month=May, UTC month=April
	ts := time.Date(2026, 5, 1, 0, 1, 0, 0, loc)
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
