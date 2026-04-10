package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestEmbed(t *testing.T) {
	embed := embed{
		Icon_:     "heatpump",
		Features_: []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice},
	}

	other := map[string]any{
		"features": []string{"switchdevice"},
	}

	require.NoError(t, util.DecodeOther(other, &embed))

	// note: slices are not merged
	require.Len(t, embed.Features_, 1)
}
