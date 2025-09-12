package fixed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZonesForDay(t *testing.T) {
	zones := Zones{
		{Days: nil},
		{Days: []Day{Monday, Tuesday, Wednesday}},
		{Days: []Day{Wednesday, Thursday, Friday}},
	}

	assert.Len(t, zones.ForDayAndMonth(Sunday, April), 1)
	assert.Len(t, zones.ForDayAndMonth(Monday, April), 2)
	assert.Len(t, zones.ForDayAndMonth(Wednesday, April), 3)
	assert.Len(t, zones.ForDayAndMonth(Thursday, April), 2)
}

func TestZonesForDayAndMonth(t *testing.T) {
	zones := Zones{
		{Days: nil, Months: nil}, // Applies to all days and months
		{Days: []Day{Monday, Tuesday, Wednesday}, Months: []Month{January, February, March, April}},
		{Days: []Day{Wednesday, Thursday, Friday}, Months: []Month{April, May, June}},
	}

	// Test for specific day and month combinations
	assert.Len(t, zones.ForDayAndMonth(Sunday, April), 1)
	assert.Len(t, zones.ForDayAndMonth(Monday, January), 2)
	assert.Len(t, zones.ForDayAndMonth(Wednesday, April), 3)
	assert.Len(t, zones.ForDayAndMonth(Thursday, May), 2)
	assert.Len(t, zones.ForDayAndMonth(Saturday, July), 1)
}

func TestZonesForMonth(t *testing.T) {
	zones := Zones{
		{Months: nil},
		{Months: []Month{January, February, March, April}},
		{Months: []Month{April, May, June}},
	}

	// Test for specific months
	assert.Len(t, zones.ForDayAndMonth(Sunday, January), 2)
	assert.Len(t, zones.ForDayAndMonth(Sunday, April), 3)
	assert.Len(t, zones.ForDayAndMonth(Sunday, July), 1)
}

func TestZonesTimeTableMarkers(t *testing.T) {
	zones := Zones{
		{Hours: TimeRange{
			From: HourMin{1, 0},
			To:   HourMin{2, 0},
		}},
		{Hours: TimeRange{
			From: HourMin{2, 0}, // make sure adjacent zones don't generate duplicate markers
			To:   HourMin{3, 0},
		}},
		{Hours: TimeRange{
			From: HourMin{4, 30},
			To:   HourMin{5, 30},
		}},
	}

	expect := []HourMin{
		{0, 0},
		{1, 0},
		{2, 0},
		{3, 0},
		{4, 0}, // 1hr intervals
		{4, 30},
		{5, 0}, // 1hr intervals
		{5, 30},
	}

	// 1hr intervals
	for hour := 6; hour < 24; hour++ {
		expect = append(expect, HourMin{hour, 0})
	}

	assert.Equal(t, expect, zones.TimeTableMarkers())
}
