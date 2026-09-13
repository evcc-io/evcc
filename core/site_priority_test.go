package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetPriorityStrategy(t *testing.T) {
	site := NewSite()

	require.NoError(t, site.SetPriorityStrategy(api.PrioritySoc))
	assert.Equal(t, api.PrioritySoc, site.GetPriorityStrategy())
	v, err := settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PrioritySoc.String(), v)

	require.NoError(t, site.SetPriorityStrategy(api.PriorityDeficit))
	assert.Equal(t, api.PriorityDeficit, site.GetPriorityStrategy())
	require.NoError(t, site.SetPriorityStrategy(api.PriorityNone))
	assert.Equal(t, api.PriorityNone, site.GetPriorityStrategy())

	require.NoError(t, site.SetPriorityStrategy(api.PrioritySoc))
	assert.Error(t, site.SetPriorityStrategy(api.PriorityStrategy(99)))
	assert.Equal(t, api.PrioritySoc, site.GetPriorityStrategy())
}

func TestSetPriorityBasis(t *testing.T) {
	site := NewSite()

	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))
	assert.Equal(t, api.PriorityBasisEnergy, site.GetPriorityBasis())
	v, err := settings.String(keys.PriorityBasis)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityBasisEnergy.String(), v)

	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisPercent))
	assert.Equal(t, api.PriorityBasisPercent, site.GetPriorityBasis())

	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))
	assert.Error(t, site.SetPriorityBasis(api.PriorityBasis(99)))
	assert.Equal(t, api.PriorityBasisEnergy, site.GetPriorityBasis())
}

func TestSetPriorityHysteresis(t *testing.T) {
	site := NewSite()
	assert.Equal(t, 3, site.GetPriorityHysteresis())

	require.NoError(t, site.SetPriorityHysteresis(5))
	assert.Equal(t, 5, site.GetPriorityHysteresis())
	v, err := settings.Int(keys.PriorityHysteresis)
	require.NoError(t, err)
	assert.Equal(t, int64(5), v)

	require.NoError(t, site.SetPriorityHysteresis(99))
	require.NoError(t, site.SetPriorityHysteresis(0))

	require.NoError(t, site.SetPriorityHysteresis(7))
	assert.Error(t, site.SetPriorityHysteresis(100))
	assert.Error(t, site.SetPriorityHysteresis(-1))
	assert.Equal(t, 7, site.GetPriorityHysteresis())
}
