package api

import (
	"sort"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

func TestRatesSort(t *testing.T) {
	clock := clock.NewMock()

	r := Rates{
		{
			Price: 1,
			Start: clock.Now(),
		},
		{
			Price: 1,
			Start: clock.Now().Add(time.Hour),
		},
	}

	sort.Sort(r)
	assert.Equal(t, clock.Now(), r[1].Start)
}
