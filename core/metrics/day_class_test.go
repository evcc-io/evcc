package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDay builds a complete day with the given night and day energy per hour (kWh)
// and an optional peak hour
func testDay(day string, night, daytime, peak float64) dayEnergy {
	d := dayEnergy{day: day, slots: 96}
	for h := range 24 {
		switch {
		case h < nightHours:
			d.hours[h] = night
		case h == 18:
			d.hours[h] = peak
		default:
			d.hours[h] = daytime
		}
	}
	return d
}

func TestClassifyDays(t *testing.T) {
	var days []dayEnergy

	// distinct non-DST dates
	date := func(i int) string {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, i).Format(time.DateOnly)
	}

	// 4 weeks normal household: ~0.35 kWh/h at night, ~0.75 kWh/h at daytime, 2 kWh evening peak (~16 kWh/day)
	for i := range 28 {
		days = append(days, testDay(date(i), 0.35, 0.75, 2.0))
	}

	// 2 weeks vacation: lower night load, no peaks (~8 kWh/day, peak below 2.5 x base load)
	for i := range 14 {
		days = append(days, testDay(date(100+i), 0.32, 0.4, 0.8))
	}

	// heavy outlier day (> 2 x median)
	days = append(days, testDay(date(200), 0.35, 2.0, 5.0))

	// departure day with morning activity is not absent
	days = append(days, testDay(date(201), 0.35, 0.4, 2.0))

	// incomplete and recovered days are not classified
	incomplete := testDay(date(202), 0.1, 0.1, 0.1)
	incomplete.slots = 92 // one hour missing
	recovered := testDay(date(203), 0.1, 0.1, 0.1)
	recovered.broken = true
	days = append(days, incomplete, recovered)

	res := classifyDays(days)

	for i := range 28 {
		assert.Equal(t, DayNormal, res[date(i)], "normal day %d", i)
	}
	for i := range 14 {
		assert.Equal(t, DayAbsent, res[date(100+i)], "absent day %d", i)
	}
	assert.Equal(t, DayOutlier, res[date(200)])
	assert.Equal(t, DayNormal, res[date(201)], "departure day")
	assert.Equal(t, DayIncomplete, res[date(202)])
	assert.Equal(t, DayIncomplete, res[date(203)])
}

func TestDayComplete(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	defer func(l *time.Location) { time.Local = l }(time.Local)
	time.Local = loc

	for _, tc := range []struct {
		day      string
		slots    int
		complete bool
	}{
		{"2026-10-14", 96, true},
		{"2026-10-14", 95, false},
		{"2026-03-29", 92, true},  // 23h DST day
		{"2026-10-25", 96, false}, // 25h DST day
		{"2026-10-25", 100, true},
	} {
		assert.Equal(t, tc.complete, dayEnergy{day: tc.day, slots: tc.slots}.complete(), "%s %d", tc.day, tc.slots)
	}
}

func TestClassifyDaysEmpty(t *testing.T) {
	assert.Empty(t, classifyDays(nil))

	// no base load: nothing to compare against
	day := time.Now().Format(time.DateOnly)
	assert.Equal(t, map[string]DayClass{day: DayNormal}, classifyDays([]dayEnergy{testDay(day, 0, 0, 0)}))
}
