package fixed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDays(t *testing.T) {
	d, err := ParseDays(" sunday ")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays("sun")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays("so")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays("0 ")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays(" 7")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Sunday}, d)

	d, err = ParseDays(" 6-7 ")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Saturday, Sunday}, d)

	d, err = ParseDays("1- 7")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}, d)

	_, err = ParseDays(" ")
	assert.NoError(t, err)
	assert.Equal(t, []Day{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}, d)

	d, err = ParseDays("1, 3-7")
	assert.NoError(t, err)
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
