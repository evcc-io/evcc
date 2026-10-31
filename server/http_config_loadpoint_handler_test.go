package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
			"mode":    "pv", // runtime setting persisted by the settings adapter
		},
	}

	var instance loadpoint.API // nil: disabled loadpoint has no live instance
	dev := config.NewConfigurableDevice(&conf, instance)

	var res loadpointFullConfig
	require.NotPanics(t, func() {
		var err error
		res, err = loadpointConfig(dev)
		require.NoError(t, err)
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
		deleteLoadpointHandler(deleteCircuitTestSite{})(rec, req)
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}

// deleting a loadpoint deletes its charger and the devices the charger owns through template params
func TestDeleteLoadpointCascadesChargerRefs(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	config.Reset()

	relay, err := config.AddConfig(templates.Charger, map[string]any{"template": "demo-switch"}, config.WithProperties(config.Properties{Type: "template"}))
	require.NoError(t, err)
	require.NoError(t, config.Chargers().Add(config.NewConfigurableDevice[api.Charger](&relay, nil)))

	charger, err := config.AddConfig(templates.Charger, map[string]any{
		"template": "sgready-relays",
		"boost":    config.NameForID(relay.ID),
	}, config.WithProperties(config.Properties{Type: "template"}))
	require.NoError(t, err)
	require.NoError(t, config.Chargers().Add(config.NewConfigurableDevice[api.Charger](&charger, nil)))

	lpConf, err := config.AddConfig(templates.Loadpoint, map[string]any{"charger": config.NameForID(charger.ID), "title": "Heat pump"})
	require.NoError(t, err)
	lp := loadpoint.NewMockAPI(gomock.NewController(t))
	lp.EXPECT().GetChargerRef().Return(config.NameForID(charger.ID)).AnyTimes()
	lp.EXPECT().GetMeterRef().Return("").AnyTimes()
	lp.EXPECT().SetChargerRef("").AnyTimes()
	require.NoError(t, config.Loadpoints().Add(config.NewConfigurableDevice[loadpoint.API](&lpConf, lp)))

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(lpConf.ID)})
	rec := httptest.NewRecorder()
	deleteLoadpointHandler(deleteCircuitTestSite{})(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	for _, id := range []int{charger.ID, relay.ID} {
		_, err := config.Chargers().ByName(config.NameForID(id))
		assert.ErrorIs(t, err, config.ErrNotFound)
	}
}
