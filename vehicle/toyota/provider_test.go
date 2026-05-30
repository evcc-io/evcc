package toyota

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderRange(t *testing.T) {
	t.Run("uses range with ac when available", func(t *testing.T) {
		p := &Provider{
			status: func() (Status, error) {
				var res Status
				res.Payload.EvRange = EvRange{Unit: "km", Value: 88}
				res.Payload.EvRangeWithAc = EvRange{Unit: "km", Value: 77}
				return res, nil
			},
		}

		rng, err := p.Range()
		require.NoError(t, err)
		require.EqualValues(t, 77, rng)
	})

	t.Run("falls back to range without ac when range with ac is missing", func(t *testing.T) {
		p := &Provider{
			status: func() (Status, error) {
				var res Status
				res.Payload.EvRange = EvRange{Unit: "km", Value: 88}
				return res, nil
			},
		}

		rng, err := p.Range()
		require.NoError(t, err)
		require.EqualValues(t, 88, rng)
	})
}
