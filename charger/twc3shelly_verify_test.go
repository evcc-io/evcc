package charger

// Template verification (no device access):
// Template registrieren, validieren und rendern. Läuft überall.
//
// LIVE verification (nur Read-only-Gerätezugriffe, kein Phasenwechsel!):
// den Charger gegen die echten Geräte (TWC3 + Shelly) bauen.
// Opt-in, weil die Adressen hardcodiert sind und CI/andere Maschinen die
// Geräte nicht erreichen:
//
//	LIVE-Test: go test ./charger/ -run TestTwc3ShellyLive -v
//	mit:       TWC3SHELLY_LIVE=1 (und TWC3_HOST/SHELLY_HOST bei Abweichung)

import (
	"os"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	yaml "go.yaml.in/yaml/v3"
)

func TestTwc3ShellyTemplate(t *testing.T) {
	// 1. Template ist über init() (embedded FS) bereits registrieren + validiert
	tmpl, err := templates.ByName(templates.Charger, "twc3shelly")
	if err != nil {
		t.Fatalf("template by name: %v", err)
	}
	if err := tmpl.Validate(); err != nil {
		t.Fatalf("template validate: %v", err)
	}
	t.Logf("template ok: %d params, capabilities=%v", len(tmpl.Params), tmpl.Capabilities)

	// 2. Rendern
	b, _, err := tmpl.RenderResult(templates.Charger, templates.RenderModeInstance, map[string]any{
		"host":   "192.168.1.20",
		"shelly": "192.168.1.100",
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
}

func TestTwc3ShellyLive(t *testing.T) {
	if os.Getenv("TWC3SHELLY_LIVE") != "1" {
		t.Skip("live device verification requires TWC3SHELLY_LIVE=1 (devices TWC3/SHELLY_HOST must be reachable)")
	}
	host := firstNonEmpty(os.Getenv("TWC3_HOST"), "192.168.178.27")
	shelly := firstNonEmpty(os.Getenv("SHELLY_HOST"), "192.168.178.167")

	// 4. Charger gegen LIVE-Geräte bauen (read-only Calls an Shelly)
	//    Die echte Pipeline (plugin/charger.go) dekodiert in config.Typed:
	//    "type" landet in .Type und wird NICHT an den Konstruktor übergeben.
	cfg := map[string]any{
		"host":   host,
		"shelly": shelly,
	}
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

	t.Log("LIVE-VERIFICATION OK (kein Phasenwechsel ausgeführt)")
}

// firstNonEmpty returns the first non-empty string or the fallback
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
