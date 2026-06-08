package eebus

import (
	"testing"

	"github.com/enbility/spine-go/util"
	"github.com/stretchr/testify/assert"
)

func TestFmtPower(t *testing.T) {
	assert.Equal(t, "n/a", fmtPower(nil))
	assert.Equal(t, "0W", fmtPower(util.Ptr(0.0)))
	assert.Equal(t, "2300W", fmtPower(util.Ptr(2300.0)))
	assert.Equal(t, "1234.5W", fmtPower(util.Ptr(1234.5)))
}
