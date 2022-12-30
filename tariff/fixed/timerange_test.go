package fixed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTimeRange(t *testing.T) {
	d, err := ParseTimeRanges(" 7:00 - 12:30 ")
	assert.NoError(t, err)
	assert.Equal(t, []TimeRange{{HourMin{7, 0}, HourMin{12, 30}}}, d, "hour:min range")

	d, err = ParseTimeRanges("8-10")
	assert.NoError(t, err)
	assert.Equal(t, []TimeRange{{HourMin{8, 0}, HourMin{10, 0}}}, d, "hour range")

	d, err = ParseTimeRanges("8-10,20-22:30")
	assert.NoError(t, err)
	assert.Equal(t, []TimeRange{
		{HourMin{8, 0}, HourMin{10, 0}},
		{HourMin{20, 0}, HourMin{22, 30}},
	}, d)
}

func TestTimeRangeContains(t *testing.T) {
	tr := TimeRange{
		From: HourMin{1, 0},
		To:   HourMin{2, 0},
	}

	assert.False(t, tr.Contains(HourMin{0, 0}))
	assert.True(t, tr.Contains(HourMin{1, 0}))
	assert.True(t, tr.Contains(HourMin{1, 59}))
	assert.False(t, tr.Contains(HourMin{2, 0}))
	assert.False(t, tr.Contains(HourMin{3, 0}))
}

func TestTimeRangeOpenEndContains(t *testing.T) {
	tr := TimeRange{
		From: HourMin{1, 0},
	}

	assert.False(t, tr.Contains(HourMin{0, 0}))
	assert.True(t, tr.Contains(HourMin{1, 0}))
	assert.True(t, tr.Contains(HourMin{1, 59}))
	assert.True(t, tr.Contains(HourMin{2, 0}))
	assert.True(t, tr.Contains(HourMin{23, 0}))
}
