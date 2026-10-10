package measurement

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombinePhases(t *testing.T) {
	val := func(f float64) func() (float64, error) { return func() (float64, error) { return f, nil } }
	fail := func(err error) func() (float64, error) { return func() (float64, error) { return 0, err } }

	t.Run("unavailable phases are zero", func(t *testing.T) {
		a, b, c, err := CombinePhases([3]func() (float64, error){val(5), fail(api.ErrNotAvailable), fail(api.ErrNotAvailable)})()
		require.NoError(t, err)
		assert.Equal(t, []float64{5, 0, 0}, []float64{a, b, c})
	})

	t.Run("all unavailable", func(t *testing.T) {
		na := fail(api.ErrNotAvailable)
		_, _, _, err := CombinePhases([3]func() (float64, error){na, na, na})()
		assert.ErrorIs(t, err, api.ErrNotAvailable)
	})

	t.Run("other errors", func(t *testing.T) {
		_, _, _, err := CombinePhases([3]func() (float64, error){val(5), fail(errors.New("fail")), val(1)})()
		assert.Error(t, err)
	})
}
