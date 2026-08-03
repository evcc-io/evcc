package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetPriorityStrategy(t *testing.T) {
	site := NewSite()

	// valid: soc
	require.NoError(t, site.SetPriorityStrategy(api.PrioritySoc))
	assert.Equal(t, api.PrioritySoc, site.GetPriorityStrategy())
	v, err := settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PrioritySoc.String(), v, "soc must be persisted")

	// valid: deficit
	require.NoError(t, site.SetPriorityStrategy(api.PriorityDeficit))
	assert.Equal(t, api.PriorityDeficit, site.GetPriorityStrategy())
	v, err = settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityDeficit.String(), v)

	// valid: none (default)
	require.NoError(t, site.SetPriorityStrategy(api.PriorityNone))
	assert.Equal(t, api.PriorityNone, site.GetPriorityStrategy())
	v, err = settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityNone.String(), v)

	// invalid: rejected, state unchanged
	require.NoError(t, site.SetPriorityStrategy(api.PrioritySoc))
	assert.Error(t, site.SetPriorityStrategy(api.PriorityStrategy(99)))
	assert.Equal(t, api.PrioritySoc, site.GetPriorityStrategy(), "invalid strategy must not change state")
	v, err = settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PrioritySoc.String(), v, "invalid strategy must not be persisted")
}

func TestSetPriorityBasis(t *testing.T) {
	site := NewSite()

	// valid: energy
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))
	assert.Equal(t, api.PriorityBasisEnergy, site.GetPriorityBasis())
	v, err := settings.String(keys.PriorityBasis)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityBasisEnergy.String(), v, "energy must be persisted")

	// valid: percent (default)
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisPercent))
	assert.Equal(t, api.PriorityBasisPercent, site.GetPriorityBasis())
	v, err = settings.String(keys.PriorityBasis)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityBasisPercent.String(), v)

	// invalid: rejected, state unchanged
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))
	assert.Error(t, site.SetPriorityBasis(api.PriorityBasis(99)))
	assert.Equal(t, api.PriorityBasisEnergy, site.GetPriorityBasis(), "invalid basis must not change state")
	v, err = settings.String(keys.PriorityBasis)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityBasisEnergy.String(), v, "invalid basis must not be persisted")
}

func TestSetPriorityHysteresis(t *testing.T) {
	site := NewSite()

	// valid
	require.NoError(t, site.SetPriorityHysteresis(5))
	assert.Equal(t, 5, site.GetPriorityHysteresis())
	v, err := settings.Int(keys.PriorityHysteresis)
	require.NoError(t, err)
	assert.Equal(t, int64(5), v, "valid hysteresis must be persisted")

	// boundary: 99 ok
	require.NoError(t, site.SetPriorityHysteresis(99))
	assert.Equal(t, 99, site.GetPriorityHysteresis())

	// boundary: 0 ok (off)
	require.NoError(t, site.SetPriorityHysteresis(0))
	assert.Equal(t, 0, site.GetPriorityHysteresis())

	// invalid: > 99 rejected, state unchanged
	require.NoError(t, site.SetPriorityHysteresis(7))
	assert.Error(t, site.SetPriorityHysteresis(100))
	assert.Equal(t, 7, site.GetPriorityHysteresis(), "out-of-range hysteresis must not change state")
	v, err = settings.Int(keys.PriorityHysteresis)
	require.NoError(t, err)
	assert.Equal(t, int64(7), v, "out-of-range hysteresis must not be persisted")

	// invalid: negative rejected
	assert.Error(t, site.SetPriorityHysteresis(-1))
	assert.Equal(t, 7, site.GetPriorityHysteresis(), "negative hysteresis must not change state")
}
