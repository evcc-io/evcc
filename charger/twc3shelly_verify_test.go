package charger

// LIVE-Verifikation (nur Read-only-Gerätezugriffe, kein Phasenwechsel!):
// Template registrieren, validieren, rendern und den Charger gegen die
// echten Geräte (TWC3 .27 + Shelly .167) bauen.
//
// Laufzeit: go test ./charger/ -run TestTwc3ShellyLive -v

import (
	"testing"

	yaml "go.yaml.in/yaml/v3"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func TestTwc3ShellyLive(t *testing.T) {
	// 1. Template ist über init() (embedded FS) bereits registriert + validiert
	tmpl, err := templates.ByName(templates.Charger, "twc3shelly")
	if err != nil {
		t.Fatalf("template by name: %v", err)
	}
	if err := tmpl.Validate(); err != nil {
		t.Fatalf("template validate: %v", err)
	}
	t.Logf("template ok: %d params, capabilities=%v", len(tmpl.Params), tmpl.Capabilities)

	// 2. Rendern mit echten Werten
	b, _, err := tmpl.RenderResult(templates.Charger, templates.RenderModeInstance, map[string]any{
		"host":   "192.168.178.27",
		"shelly": "192.168.178.167",
	})
	if err != nil {
		t.Fatalf("template render: %v", err)
	}
	t.Logf("gerendertes YAML:\n%s", string(b))

	// 3. Gerendertes YAML in Config-Map decodieren
	var cfg map[string]any
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("rendered yaml decode: %v", err)
	}
	for k, v := range cfg {
		t.Logf("config %s = %v", k, v)
	}
	if cfg["type"] != "twc3shelly" {
		t.Fatalf("unexpected type: %v", cfg["type"])
	}

	// 4. Charger gegen LIVE-Geräte bauen (read-only Calls an Shelly)
	//    Die echte Pipeline (plugin/charger.go) dekodiert in config.Typed:
	//    "type" landet in .Type und wird NICHT an den Konstruktor übergeben.
	delete(cfg, "type")
	ch, err := NewTwc3ShellyFromConfig(cfg)
	if err != nil {
		t.Fatalf("charger construct: %v", err)
	}
	t.Log("charger constructed ok (Shelly-Connectivity bestätigt)")

	// 5. Interfaces
	_, hasPS := api.Cap[api.PhaseSwitcher](ch)
	_, hasMeter := api.Cap[api.Meter](ch)
	_, hasStatus := api.Cap[api.Charger](ch)
	t.Logf("interfaces: PhaseSwitcher=%v Meter=%v Charger=%v", hasPS, hasMeter, hasStatus)
	if !hasPS {
		t.Fatal("charger does NOT implement api.PhaseSwitcher")
	}

	// 6. Read-only Status
	st, err := ch.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	t.Logf("TWC3 status: %v", st)

	m, ok := api.Cap[api.Meter](ch)
	if !ok {
		t.Fatal("charger does NOT implement api.Meter")
	}
	p, err := m.CurrentPower()
	if err != nil {
		t.Fatalf("power: %v", err)
	}
	t.Logf("TWC3 power: %.1f W", p)

	t.Log("LIVE-VERIFIKATION OK (kein Phasenwechsel ausgeführt)")
}
