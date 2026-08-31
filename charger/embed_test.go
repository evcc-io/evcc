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

func TestEmbedPredictor(t *testing.T) {
	embed := embed{
		Icon_:     "heatpump",
		Features_: []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice},
	}

	other := map[string]any{
		"predictor": []string{"demandtemperature"},
	}

	require.NoError(t, util.DecodeOther(other, &embed))

	// features are unchanged, predictor is appended via Features()
	require.Equal(t, []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice}, embed.Features_)
	require.Equal(t, []api.Feature{api.DemandTemperature}, embed.Predictor_)
	require.Contains(t, embed.Features(), api.DemandTemperature)
	require.Contains(t, embed.Features(), api.Heating)
}
