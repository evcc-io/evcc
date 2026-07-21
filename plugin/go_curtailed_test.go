package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurtailedScripts verifies the curtailed percent scripts used by the meter
// templates evaluate to an int64 for the given register readings.
func TestCurtailedScripts(t *testing.T) {
	// huawei-sun2000-hybrid, huawei-emma
	modeScript := `
limit := percent
if mode == 0 {
  limit = 100
}
limit`

	// sunspec-*-curtailable, kostal-plenticore-gen2, enphase-modbus
	enaScript := `
percent := 100
if ena {
  percent = int(limit)
}
percent`

	// sungrow-ihm
	sungrowScript := `
percent := 100
if ena != 0 {
  percent = ratio
}
percent`

	// atmoce
	atmoceScript := `
percent := 100
if limit != 0xFFFFFFFF && maxacpower > 0 {
  percent = limit * 100 / maxacpower
}
percent`

	for _, tc := range []struct {
		name   string
		script string
		in     map[string]any
		want   int64
	}{
		{"mode uncurtailed", modeScript, map[string]any{"mode": 0, "percent": 60}, 100},
		{"mode curtailed", modeScript, map[string]any{"mode": 7, "percent": 60}, 60},
		{"ena disabled", enaScript, map[string]any{"ena": false, "limit": 60.0}, 100},
		{"ena enabled", enaScript, map[string]any{"ena": true, "limit": 60.0}, 60},
		{"ena enabled fractional", enaScript, map[string]any{"ena": true, "limit": 59.5}, 59},
		{"sungrow disabled", sungrowScript, map[string]any{"ena": 0, "ratio": 60}, 100},
		{"sungrow enabled", sungrowScript, map[string]any{"ena": 1, "ratio": 30}, 30},
		{"atmoce unlimited", atmoceScript, map[string]any{"limit": 0xFFFFFFFF, "maxacpower": 10000}, 100},
		{"atmoce limited", atmoceScript, map[string]any{"limit": 3000, "maxacpower": 10000}, 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := make([]map[string]any, 0, len(tc.in))
			for name, val := range tc.in {
				typ := "int"
				switch val.(type) {
				case bool:
					typ = "bool"
				case float64:
					typ = "float"
				}

				in = append(in, map[string]any{
					"name":   name,
					"type":   typ,
					"config": map[string]any{"source": "const", "value": val},
				})
			}

			p, err := NewGoPluginFromConfig(context.TODO(), map[string]any{
				"script": tc.script,
				"in":     in,
			})
			require.NoError(t, err)

			get, err := p.(IntGetter).IntGetter()
			require.NoError(t, err)

			res, err := get()
			require.NoError(t, err)
			assert.Equal(t, tc.want, res)
		})
	}
}
