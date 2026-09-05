package plugin

import "testing"

func TestSolarmanFromConfig(t *testing.T) {
	provider, err := NewSolarmanFromConfig(t.Context(), map[string]any{
		"host":   "192.0.2.2",
		"serial": 3875738533,
		"register": map[string]any{
			"address": 86,
			"type":    "holding",
			"decode":  "uint32s",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := provider.(*Solarman).FloatGetter(); err != nil {
		t.Fatal(err)
	}
}
