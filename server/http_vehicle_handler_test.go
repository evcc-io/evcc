package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// REST API Integration Test for Pause and Resume Endpoints
func TestPauseRepeatingPlansHandler(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	mockVehicle := api.NewMockVehicle(ctrl)
	mockVehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	mockVehicle.EXPECT().GetTitle().Return("Test EV4").AnyTimes()
	mockVehicle.EXPECT().Icon().Return("").AnyTimes()
	mockVehicle.EXPECT().Capacity().Return(float64(60)).AnyTimes()
	mockVehicle.EXPECT().Phases().Return(3).AnyTimes()
	mockVehicle.EXPECT().Features().Return(nil).AnyTimes()

	const name = "ev4"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(mockVehicle)),
	))

	site := core.NewSite()
	srv := NewHTTPd("", nil, Customization{})
	srv.RegisterSiteHandlers(site)

	router := srv.Router()

	t.Run("POST valid pause timestamp", func(t *testing.T) {
		// Dynamic future timestamp (e.g. 2026-08-20T12:00:00Z)
		future := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
		req := httptest.NewRequest(http.MethodPost, "/api/vehicles/ev4/plan/pause/"+future.Format(time.RFC3339), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var res struct {
			PausedUntil *time.Time `json:"pausedUntil"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
		require.NotNil(t, res.PausedUntil)
		assert.True(t, res.PausedUntil.Equal(future))

		v, err := site.Vehicles().ByName("ev4")
		require.NoError(t, err)
		assert.True(t, v.GetPausedUntil().Equal(future))
	})

	t.Run("POST invalid pause timestamp", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/vehicles/ev4/plan/pause/invalid-timestamp", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("POST past pause timestamp rejected", func(t *testing.T) {
		v, err := site.Vehicles().ByName("ev4")
		require.NoError(t, err)
		prevPausedUntil := v.GetPausedUntil()

		req := httptest.NewRequest(http.MethodPost, "/api/vehicles/ev4/plan/pause/2020-01-01T00:00:00Z", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, prevPausedUntil, v.GetPausedUntil())
	})

	t.Run("DELETE resume repeating plans", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/vehicles/ev4/plan/pause", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		v, err := site.Vehicles().ByName("ev4")
		require.NoError(t, err)
		assert.True(t, v.GetPausedUntil().IsZero())
	})
}
