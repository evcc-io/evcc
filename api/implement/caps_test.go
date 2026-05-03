package implement

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

type isCapable struct {
	Capabilities
}

func TestImplement(t *testing.T) {
	s := &isCapable{
		Capabilities: Caps(),
	}

	Implements(s, Meter(func() (float64, error) {
		return 0, nil
	}))

	mm, ok := api.Cap[api.Meter](s)
	assert.True(t, ok)

	v, err := mm.CurrentPower()
	assert.NoError(t, err)
	assert.Equal(t, 0.0, v)
}
