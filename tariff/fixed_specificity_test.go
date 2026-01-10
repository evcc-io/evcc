package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			{0.20, "0-5", ""},                // general (all year)
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

	for _, tc := range testCases {
		clock := clock.NewMock()
		at.(*Fixed).clock = clock
		clock.Set(time.Date(2025, tc.month, 15, 3, 0, 0, 0, time.UTC))

		rr, err := at.Rates()
		require.NoError(t, err)

		r, err := rr.At(clock.Now())
		require.NoError(t, err)

		assert.Equal(t, tc.expected, r.Value,
			"TZ=%s, %s: expected %.2f", time.UTC, tc.month, tc.expected)
	}
}

func TestPartiallyOverlappingMonths(t *testing.T) {
	at, err := NewFixedFromConfig(map[string]any{
		"price": 0.0,
		"zones": []struct {
			Price  float64
			Hours  string
			Months string
		}{
			{0.10, "0-5", "Jan"},
			{0.20, "0-5", "Feb"},
			{0.30, "0-5", "Jan-Mar"},
		},
	})
	require.NoError(t, err)

	clock := clock.NewMock()
	tf := at.(*Fixed)
	tf.clock = clock

	// Test for January → should detect ambiguity (Zone 0 + Zone 2)
	clock.Set(time.Date(2025, time.January, 10, 1, 0, 0, 0, time.UTC))
	_, _ = tf.Rates() // triggers warning

	// Test for February → should detect ambiguity (Zone 1 + Zone 2)
	clock.Set(time.Date(2025, time.February, 10, 1, 0, 0, 0, time.UTC))
	_, _ = tf.Rates() // triggers warning

	// Test for March → only Zone 2 applies → no warning
	clock.Set(time.Date(2025, time.March, 10, 1, 0, 0, 0, time.UTC))
	_, _ = tf.Rates()
}
