package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/require"
)

func TestCacheInterval(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	p := &cachingProxy{key: "test", interval: time.Hour}
	require.NoError(t, p.cachePut(api.TariffTypeSolar, api.Rates{
		{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 1},
	}))

	// within interval
	res, err := p.cacheGet()
	require.NoError(t, err)
	require.Equal(t, api.TariffTypeSolar, res.Type)

	// interval elapsed
	p.cached.Updated = time.Now().Add(-2 * time.Hour)
	_, err = p.cacheGet()
	require.Error(t, err)
}
