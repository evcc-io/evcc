package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGridPeak(t *testing.T) {
	var p gridPeak
	start := time.Date(2026, 10, 31, 23, 7, 0, 0, time.UTC)

	// partial interval after start is not recorded
	assert.False(t, p.add(start, 8000))
	assert.False(t, p.add(start.Add(8*time.Minute), 4000))
	assert.Equal(t, 0.0, p.peak(start))

	// 23:15-23:30: 10 min at 4 kW, 5 min at 1 kW
	assert.False(t, p.add(start.Add(18*time.Minute), 1000))
	assert.True(t, p.add(start.Add(23*time.Minute), 1000))
	assert.InDelta(t, 3000, p.peak(start), 1e-6)

	// export does not count, lower average does not raise
	assert.False(t, p.add(start.Add(30*time.Minute), -5000))
	assert.False(t, p.add(start.Add(38*time.Minute), 5000))
	assert.InDelta(t, 3000, p.peak(start), 1e-6)

	// 23:45-00:00 belongs to october
	assert.True(t, p.add(start.Add(53*time.Minute), 0))
	assert.InDelta(t, 5000, p.peak(start), 1e-6)

	// new month starts without peak
	nov := start.Add(53 * time.Minute)
	assert.Equal(t, 0.0, p.peak(nov))

	// gap invalidates the interval
	assert.False(t, p.add(nov.Add(time.Hour), 2000))
	assert.False(t, p.add(nov.Add(time.Hour+8*time.Minute), 2000))
	assert.Equal(t, 0.0, p.peak(nov))

	// 01:15-01:30 complete
	assert.False(t, p.add(nov.Add(time.Hour+23*time.Minute), 2000))
	assert.True(t, p.add(nov.Add(time.Hour+30*time.Minute), 2000))
	assert.InDelta(t, 2000, p.peak(nov), 1e-6)
	assert.Equal(t, "2026-11", p.Month)
}
