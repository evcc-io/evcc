package server

import (
	"net/http"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
)

// LiveConfigHandler delivers the current live configuration (file + UI/DB) as JSON
func LiveConfigHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Site-Config
		siteRes := struct {
			Title   string   `json:"title"`
			Grid    string   `json:"grid"`
			PV      []string `json:"pv"`
			Battery []string `json:"battery"`
			Aux     []string `json:"aux"`
			Ext     []string `json:"ext"`
		}{
			Title:   site.GetTitle(),
			Grid:    site.GetGridMeterRef(),
			PV:      site.GetPVMeterRefs(),
			Battery: site.GetBatteryMeterRefs(),
			Aux:     site.GetAuxMeterRefs(),
			Ext:     site.GetExtMeterRefs(),
		}

		// aux/ext als leere Arrays statt null, falls nicht definiert
		if siteRes.Aux == nil {
			siteRes.Aux = []string{}
		}
		if siteRes.Ext == nil {
			siteRes.Ext = []string{}
		}

		// Loadpoints
		loadpoints := []any{}
		for _, dev := range config.Loadpoints().Devices() {
			loadpoints = append(loadpoints, loadpointConfig(dev))
		}

		// Charger
		chargers, _ := devicesConfig(templates.Charger, config.Chargers())
		// Vehicle
		vehicles, _ := devicesConfig(templates.Vehicle, config.Vehicles())
		// Meter
		meters, _ := devicesConfig(templates.Meter, config.Meters())

		// Globale YAML/JSON-Blöcke
		// Zugriff über server/db/settings
		importSettings := func(key string) any {
			val, _ := settings.String(key)
			return val
		}
		importJson := func(key string, obj any) any {
			_ = settings.Json(key, &obj)
			return obj
		}

		// Tariffs als Objekt laden (nicht als String)
		tariffs := map[string]any{}
		_ = settings.Json("tariffs", &tariffs)

		res := map[string]any{
			"site":         siteRes,
			"loadpoints":   loadpoints,
			"chargers":     chargers,
			"vehicles":     vehicles,
			"meters":       meters,
			"eebus":        importSettings("eebus"),
			"hems":         importSettings("hems"),
			"tariffs":      tariffs,
			"messaging":    importSettings("messaging"),
			"modbusproxy":  importSettings("modbusproxy"),
			"circuits":     importSettings("circuits"),
			"network":      importJson("network", map[string]any{}),
			"mqtt":         importJson("mqtt", map[string]any{}),
			"influx":       importJson("influx", map[string]any{}),
			"interval":     importSettings("interval"),
			"sponsor":      importSettings("sponsor"),
			"sponsorToken": importSettings("sponsorToken"),
			"version":      importSettings("version"),
			"fatal":        importSettings("fatal"),
			"startup":      importSettings("startup"),
			"plant":        importSettings("plant"),
			"telemetry":    importSettings("telemetry"),
		}
		jsonResult(w, res)
	}
}
