package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParam(t *testing.T) {
	lp := 2
	p := Param{
		Key: "power",
		Val: 4711,
	}
	assert.Equal(t, "power", p.UniqueID())

	p.Loadpoint = &lp
	assert.Equal(t, "2.power", p.UniqueID())
}
