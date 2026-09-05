package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// huaweiCurtailedReader renders the pv usage of the huawei-sun2000-hybrid
// template and returns its curtailed reader with the modbus inputs replaced
// by the given constants, keyed by input name.
func huaweiCurtailedReader(t *testing.T, inputs map[string]any) func() (int64, error) {
	t.Helper()

	tmpl, err := templates.ByName(templates.Meter, "huawei-sun2000-hybrid")
	require.NoError(t, err)

	b, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, map[string]any{
		"usage":      "pv",
		"modbus":     "tcpip",
		"host":       "127.0.0.1",
		"port":       502,
		"maxacpower": 11000,
	})
	require.NoError(t, err)

	var rendered struct {
		Curtailed map[string]any `yaml:"curtailed"`
	}
	require.NoError(t, yaml.Unmarshal(b, &rendered))
	require.NotEmpty(t, rendered.Curtailed, "curtailed reader missing")

	in, ok := rendered.Curtailed["in"].([]any)
	require.True(t, ok, "curtailed reader has no inputs")

	seen := map[string]bool{}
	for _, i := range in {
		input := i.(map[string]any)
		name := input["name"].(string)
		value, ok := inputs[name]
		require.True(t, ok, "no test value for input %s", name)
		input["config"] = map[string]any{"source": "const", "value": value}
		seen[name] = true
	}
	for name := range inputs {
		require.True(t, seen[name], "template has no input %s", name)
	}

	delete(rendered.Curtailed, "source")

	p, err := plugin.NewGoPluginFromConfig(context.Background(), rendered.Curtailed)
	require.NoError(t, err)

	g, err := p.(plugin.IntGetter).IntGetter()
	require.NoError(t, err)

	return g
}

// The curtailed reader must map every active power control mode (47415) to
// the feed-in limit as percent of nominal, not just the percentage mode 7.
func TestHuaweiSun2000HybridCurtailed(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		mode, percent, watt, rated int
		want                       int64
	}{
		{"unlimited", 0, 0, 0, 11000, 100},
		{"di scheduling is unknown, treat as uncurtailed", 1, 0, 0, 11000, 100},
		{"zero export", 5, 0, 0, 11000, 0},
		{"kw limit relative to nominal", 6, 0, 6300, 11000, 57},
		{"kw limit above nominal clamps to 100", 6, 0, 12000, 11000, 100},
		{"kw limit without nominal is uncurtailed", 6, 0, 6300, 0, 100},
		{"negative kw limit clamps to 0", 6, 0, -1000, 11000, 0},
		{"percent limit", 7, 63, 0, 11000, 63},
		{"percent limit ignores stale kw register", 7, 63, 6300, 11000, 63},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := huaweiCurtailedReader(t, map[string]any{
				"mode":    tc.mode,
				"percent": tc.percent,
				"watt":    tc.watt,
				"rated":   tc.rated,
			})

			got, err := g()
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
