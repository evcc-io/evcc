package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadpointConfigDisabledNilInstance ensures loadpointConfig does not panic
// for a disabled loadpoint whose instance was never created (nil loadpoint.API).
func TestLoadpointConfigDisabledNilInstance(t *testing.T) {
	conf := config.Config{
		Class:      templates.Loadpoint,
		Properties: config.Properties{Disable: true},
		Data: map[string]any{
			"charger": "wallbox",
			"meter":   "lp-meter",
			"title":   "Garage",
		},
	}

	var instance loadpoint.API // nil: disabled loadpoint has no live instance
	dev := config.NewConfigurableDevice(&conf, instance)

	var res loadpointFullConfig
	require.NotPanics(t, func() {
		res = loadpointConfig(dev)
	})

	assert.True(t, res.Disable)
	assert.Equal(t, "wallbox", res.Charger)
	assert.Equal(t, "lp-meter", res.Meter)
}

// TestDeleteLoadpointDisabledNilInstance ensures deleteLoadpointHandler does not
// dereference a nil instance when deleting a disabled loadpoint.
func TestDeleteLoadpointDisabledNilInstance(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	conf, err := config.AddConfig(templates.Loadpoint,
		map[string]any{"charger": "wallbox", "title": "Garage"},
		config.WithProperties(config.Properties{Disable: true}),
	)
	require.NoError(t, err)

	var instance loadpoint.API // nil: disabled loadpoint has no live instance
	dev := config.NewConfigurableDevice(&conf, instance)
	require.NoError(t, config.Loadpoints().Add(dev))
	t.Cleanup(func() { _ = config.Loadpoints().Delete(config.NameForID(conf.ID)) })

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(conf.ID)})
	rec := httptest.NewRecorder()

	require.NotPanics(t, func() {
		deleteLoadpointHandler()(rec, req)
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}
