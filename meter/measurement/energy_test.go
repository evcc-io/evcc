package measurement

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonZeroEnergy(t *testing.T) {
	assert.Nil(t, nonZeroEnergy(nil), "nil getter stays nil")

	// non-zero value passes through
	f, err := nonZeroEnergy(func() (float64, error) { return 1234.5, nil })()
	require.NoError(t, err)
	assert.Equal(t, 1234.5, f)

	// zero value is reported as not available
	_, err = nonZeroEnergy(func() (float64, error) { return 0, nil })()
	assert.ErrorIs(t, err, api.ErrNotAvailable)

	// upstream error is preserved
	upstream := errors.New("boom")
	_, err = nonZeroEnergy(func() (float64, error) { return 0, upstream })()
	assert.ErrorIs(t, err, upstream)
}
