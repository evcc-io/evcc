package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPatchLoadpointStaticConfig(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(strconv.FormatBool(live), func(t *testing.T) {
			require.NoError(t, db.NewInstance("sqlite", ":memory:"))
			data := map[string]any{
				"title": "Garage", "charger": "wallbox", "circuit": "old",
				"minSoc": 35, "priority": 4, "solarShare": 0.6,
			}
			conf, err := config.AddConfig(templates.Loadpoint, data, config.WithProperties(config.Properties{Disable: !live}))
			require.NoError(t, err)

			var instance loadpoint.API
			if live {
				instance = loadpoint.NewMockAPI(gomock.NewController(t))
			}
			dev := config.NewConfigurableDevice(&conf, instance)
			require.NoError(t, config.Loadpoints().Add(dev))
			t.Cleanup(func() { _ = config.Loadpoints().Delete(config.NameForID(conf.ID)) })

			patch := func(body string) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))
				req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(conf.ID)})
				rec := httptest.NewRecorder()
				patchLoadpointHandler()(rec, req)
				return rec
			}

			for _, circuit := range []string{"new", ""} {
				rec := patch(`{"circuit":"` + circuit + `"}`)
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

				data["circuit"] = circuit
				assert.Equal(t, data, dev.Config().Other)
				assert.Equal(t, !live, dev.Properties().Disable)
				assert.Equal(t, instance, dev.Instance())

				var stored config.Config
				require.NoError(t, db.Instance.First(&stored, conf.ID).Error)
				assert.Equal(t, circuit, stored.Data["circuit"])
				assert.EqualValues(t, 35, stored.Data["minSoc"])
			}

			for _, body := range []string{
				``, `[]`, `{"circuit":12}`,
				`{"circuit":{}}`, `{"circuit":true}`, `{"circuit":"new","priority":2}`,
				`{"unknown":"value"}`, `{"charger":"new-wallbox"}`,
			} {
				assert.Equal(t, http.StatusBadRequest, patch(body).Code, body)
				assert.Equal(t, data, dev.Config().Other)
				var stored config.Config
				require.NoError(t, db.Instance.First(&stored, conf.ID).Error)
				assert.Equal(t, "", stored.Data["circuit"])
				assert.EqualValues(t, 35, stored.Data["minSoc"])
			}

			for _, body := range []string{`{}`, `null`, `{"circuit":null}`} {
				require.Equal(t, http.StatusOK, patch(body).Code, body)
				assert.Equal(t, data, dev.Config().Other)
			}
		})
	}
}

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
		deleteLoadpointHandler()(rec, req)
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}
