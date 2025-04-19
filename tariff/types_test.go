package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

func TestFromTo(t *testing.T) {
	tc := []struct {
		from, to, now int
		active        bool
	}{
		{0, 0, 0, true},
		{1, 2, 1, true},
		{1, 2, 2, true},
		{1, 2, 0, false},
		{1, 2, 3, false},
		{22, 2, 21, false},
		{22, 2, 22, true},
		{22, 2, 2, true},
		{22, 2, 3, false},
	}

	for _, tc := range tc {
		clock := clock.NewMock()
		clock.Add(time.Duration(tc.now) * time.Hour)

		ft := FromTo{tc.from, tc.to}
		assert.Equal(t, tc.active, ft.IsActive(tc.now), "expected %v")
	}
}
