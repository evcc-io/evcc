package fixed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDays(t *testing.T) {
	d, err := ParseDays(" sunday ")
	require.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays("sun")
	require.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays("so")
	require.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays("0 ")
	require.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays(" 7")
	require.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays(" 6-7 ")
	require.NoError(t, err)
	assert.Equal(t, []Day{Saturday, Sunday}, d)

	d, err = ParseDays("1- 7")
	require.NoError(t, err)
	assert.Equal(t, []Day{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}, d)

	_, err = ParseDays(" ")
	require.NoError(t, err)
	assert.Equal(t, []Day{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}, d)

	d, err = ParseDays("1, 3-7")
	require.NoError(t, err)
	assert.Equal(t, []Day{Monday, Wednesday, Thursday, Friday, Saturday, Sunday}, d)

	_, err = ParseDays("-")
	assert.EqualError(t, err, "invalid day: ")

	_, err = ParseDays("-1")
	assert.EqualError(t, err, "invalid day: ")

	_, err = ParseDays(" 8 ")
	assert.EqualError(t, err, "invalid day: 8")

	_, err = ParseDays("1, 1")
	assert.EqualError(t, err, "duplicate days")

	_, err = ParseDays("0,1,2,3,4,5,6,7")
	assert.EqualError(t, err, "too many days")
}
