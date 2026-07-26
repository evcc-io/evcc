package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/logstash"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceLogHandler(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	conf, err := config.AddConfig(templates.Meter, map[string]any{},
		config.WithProperties(config.Properties{Title: "Garage"}))
	require.NoError(t, err)

	for _, area := range []string{conf.LogArea(), "other"} {
		logstash.DefaultHandler.Add(logstash.Entry{
			Time: time.Now(), Area: area, Level: slog.LevelInfo, Message: area + " entry",
		})
	}

	get := func(class string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/?level=trace", nil)
		r = mux.SetURLVars(r, map[string]string{"class": class, "id": strconv.Itoa(conf.ID)})

		w := httptest.NewRecorder()
		deviceLogHandler(w, r)
		return w
	}

	// only the device's own entries
	w := get("meter")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), conf.LogArea()+" entry")
	assert.NotContains(t, w.Body.String(), "other entry")

	// class must match the stored device
	assert.Equal(t, http.StatusNotFound, get("charger").Code)
}
