package charger

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSgReadyRelayFromConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	boost := api.NewMockCharger(ctrl)
	dim := api.NewMockCharger(ctrl)

	config.Reset()
	t.Cleanup(config.Reset)
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice[api.Charger](config.Named{Name: "boost-relay"}, boost)))
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice[api.Charger](config.Named{Name: "dim-relay"}, dim)))

	registry.AddCtx("test-relay", func(context.Context, map[string]any) (api.Charger, error) {
		return dim, nil
	})

	t.Run("referenced relays", func(t *testing.T) {
		wb, err := NewSgReadyRelayFromConfig(ctx, map[string]any{
			"boost": "boost-relay",
			"dim":   "dim-relay",
		})
		require.NoError(t, err)

		dim.EXPECT().Enable(false)
		boost.EXPECT().Enable(true)
		require.NoError(t, wb.Enable(true))
	})

	t.Run("inline dim config", func(t *testing.T) {
		wb, err := NewSgReadyRelayFromConfig(ctx, map[string]any{
			"boost": "boost-relay",
			"dim":   map[string]any{"type": "test-relay"},
		})
		require.NoError(t, err)

		boost.EXPECT().Enable(false)
		dim.EXPECT().Enable(true)
		require.NoError(t, wb.(api.Dimmer).Dim(true))
	})

	t.Run("unknown reference", func(t *testing.T) {
		_, err := NewSgReadyRelayFromConfig(ctx, map[string]any{"boost": "missing"})
		assert.ErrorIs(t, err, config.ErrNotFound)
	})
}
