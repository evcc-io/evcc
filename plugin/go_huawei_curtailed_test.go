package plugin

import (
	"testing"

	"github.com/evcc-io/evcc/plugin/golang"
	"github.com/stretchr/testify/require"
	"github.com/traefik/yaegi/interp"
)

// huaweiCurtailedScript mirrors the curtailed logic in
// templates/definition/meter/huawei-sun2000-hybrid.yaml
const huaweiCurtailedScript = `mode != 0 && !(mode == 7 && percent == 100)`

func TestHuaweiCurtailed(t *testing.T) {
	tc := []struct {
		mode, percent int64
		curtailed     bool
	}{
		{0, 100, false}, // unlimited
		{0, 50, false},  // unlimited (percent irrelevant when mode 0)
		{7, 100, false}, // percentage-limit enabled but at 100% -> not actually curtailing
		{7, 70, true},   // percentage-limit at 70%
		{7, 0, true},    // percentage-limit at 0%
		{1, 100, true},  // other active mode
		{6, 100, true},  // kW-limit at max: not special-cased, reported as curtailed
	}

	for _, tc := range tc {
		mode, percent := tc.mode, tc.percent
		p := &Go{
			vm:     func() (*interp.Interpreter, error) { return golang.RegisteredVM("", "") },
			script: huaweiCurtailedScript,
			in: []inputTransformation{
				{name: "mode", function: func() (any, error) { return mode, nil }},
				{name: "percent", function: func() (any, error) { return percent, nil }},
			},
		}

		g, err := p.BoolGetter()
		require.NoError(t, err)

		v, err := g()
		require.NoError(t, err)
		require.Equalf(t, tc.curtailed, v, "mode=%d percent=%d", tc.mode, tc.percent)
	}
}
