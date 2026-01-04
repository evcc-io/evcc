package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFixedSpecificity validates that MoreSpecific logic is necessary.
// Edge case: zones with IDENTICAL hours but different month/day constraints.
// Without MoreSpecific, the result would depend on undefined sort order.
func TestFixedSpecificity(t *testing.T) {
	at, err := NewFixedFromConfig(map[string]any{
		"price": 0.30,
		"zones": []struct {
			Price  float64
			Hours  string
			Months string
		}{
			// REVERSED: specific first, general second
			// Without MoreSpecific, this will fail!
			{0.10, "0-5", "Jan-Mar,Oct-Dec"}, // specific (winter only)
			{0.20, "0-5", ""},              // general (all year)
		},
	})
	require.NoError(t, err)

	testCases := []struct {
		month    time.Month
		expected float64
	}{
		{time.January, 0.10},  // winter: specific zone wins
		{time.June, 0.20},     // summer: general zone
		{time.December, 0.10}, // winter: specific zone wins
	}

	// Test both UTC and Local timezones
	timezones := []*time.Location{time.UTC, time.Local}
	for _, tz := range timezones {
		t.Run(tz.String(), func(t *testing.T) {
			for _, tc := range testCases {
				clock := clock.NewMock()
				at.(*Fixed).clock = clock
				clock.Set(time.Date(2025, tc.month, 15, 3, 0, 0, 0, tz))

				rr, err := at.Rates()
				require.NoError(t, err)

				r, err := rr.At(clock.Now())
				require.NoError(t, err)

				assert.Equal(t, tc.expected, r.Value,
					"TZ=%s, %s: expected %.2f", tz, tc.month, tc.expected)
			}
		})
	}
}
