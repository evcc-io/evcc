package implement

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

type isCapable struct {
	Caps
}

func TestHas(t *testing.T) {
	s := &isCapable{
		Caps: New(),
	}

	Has(s, Meter(func() (float64, error) {
		return 0, nil
	}))

	mm, ok := api.Cap[api.Meter](s)
	assert.True(t, ok)

	v, err := mm.CurrentPower()
	assert.NoError(t, err)
	assert.Equal(t, 0.0, v)
}

func TestHasPanicsOnNil(t *testing.T) {
	s := &isCapable{
		Caps: New(),
	}

	assert.Panics(t, func() {
		Has[api.Meter](s, nil)
	})
}

func TestMayIgnoresNil(t *testing.T) {
	s := &isCapable{
		Caps: New(),
	}

	assert.NotPanics(t, func() {
		May[api.Meter](s, nil)
	})

	_, ok := api.Cap[api.Meter](s)
	assert.False(t, ok)
}

func TestMayRegistersNonNil(t *testing.T) {
	s := &isCapable{
		Caps: New(),
	}

	May(s, Meter(func() (float64, error) {
		return 1.0, nil
	}))

	mm, ok := api.Cap[api.Meter](s)
	assert.True(t, ok)

	v, err := mm.CurrentPower()
	assert.NoError(t, err)
	assert.Equal(t, 1.0, v)
}

func TestMayIgnoresNilFuncConstructor(t *testing.T) {
	s := &isCapable{
		Caps: New(),
	}

	var fn func() (float64, error)
	May(s, Meter(fn))

	_, ok := api.Cap[api.Meter](s)
	assert.False(t, ok)
}
