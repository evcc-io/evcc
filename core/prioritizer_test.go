package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrioritzer(t *testing.T) {
	p := Prioritizer{demand: make(map[int]float64)}

	reduceBy := p.Prioritize(1, 1e3)
	assert.Equal(t, 0.0, reduceBy)

	reduceBy = p.Prioritize(0, 2e3)
	assert.Equal(t, 1e3, reduceBy)
}
