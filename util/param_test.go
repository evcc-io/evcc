package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParam(t *testing.T) {
	lp := 2
	sub := ParamSub{
		SubKey: "pv",
		Index:  1,
	}
	p := Param{
		Key: "power",
		Val: 4711,
	}
	assert.Equal(t, "power", p.UniqueID())

	p.LoadPoint = &lp
	assert.Equal(t, "2.power", p.UniqueID())

	p.Sub = &sub
	assert.Equal(t, "2.pv.1.power", p.UniqueID())
}
