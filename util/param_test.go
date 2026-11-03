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

func TestParamCache(t *testing.T) {
	NewParamCache().Add("foo", Param{})
}

func TestParamCacheSnapshot(t *testing.T) {
	in := make(chan Param)
	go NewParamCache().Run(in)

	in <- Param{Key: "before", Val: 1}

	res := make(chan []Param, 1)
	in <- Param{Val: Snapshot(func(state []Param) { res <- state })}

	// published after the snapshot request, must not be included
	in <- Param{Key: "after", Val: 2}

	state := <-res
	assert.Len(t, state, 1)
	assert.Equal(t, "before", state[0].Key)
}
