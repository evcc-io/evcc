package planner

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestCapacityLedger(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	next := start.Add(time.Hour)

	l := NewCapacityLedger(11000, time.Hour)

	// full budget initially
	assert.Equal(t, 11000.0, l.Available(start))

	// reserve 7400W in the first slot- the remainder is below a 3p 6A (4140W)
	// session but still fits a 1p 6A (1380W) one
	l.Reserve(api.Rates{{Start: start, End: next}}, 7400)
	assert.Equal(t, 3600.0, l.Available(start))

	// the next slot is untouched
	assert.Equal(t, 11000.0, l.Available(next))

	// over-reserving never goes negative
	l.Reserve(api.Rates{{Start: start, End: next}}, 9000)
	assert.Equal(t, 0.0, l.Available(start))
}
