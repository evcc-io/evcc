package settings

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	v := "foo"
	SetString("string", v)
	res, err := String("string")
	assert.Nil(t, err)
	assert.Equal(t, v, res)
}

func TestInt(t *testing.T) {
	v := int64(math.MaxInt64)
	SetInt("int64", v)
	res, err := Int("int64")
	assert.Nil(t, err)
	assert.Equal(t, v, res)
}

func TestFloat(t *testing.T) {
	v := 3.141
	SetFloat("float64", v)
	res, err := Float("float64")
	assert.Nil(t, err)
	assert.Equal(t, v, res)
}
