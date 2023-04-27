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

	assert.Len(t, zones.ForDay(Sunday), 1)
	assert.Len(t, zones.ForDay(Monday), 2)
	assert.Len(t, zones.ForDay(Wednesday), 3)
	assert.Len(t, zones.ForDay(Thursday), 2)
}

func TestZonesTimeTableMarkers(t *testing.T) {
	zones := Zones{
		{Hours: TimeRange{
			From: HourMin{1, 0},
			To:   HourMin{2, 0},
		}},
		{Hours: TimeRange{
			From: HourMin{3, 30},
			To:   HourMin{4, 30},
		}},
	}

	expect := []HourMin{
		{0, 0},
		{1, 0},
		{2, 0},
		{3, 0}, // 1hr intervals
		{3, 30},
		{4, 0}, // 1hr intervals
		{4, 30},
	}

	// 1hr intervals
	for hour := 5; hour < 24; hour++ {
		expect = append(expect, HourMin{hour, 0})
	}

	assert.Equal(t, expect, zones.TimeTableMarkers())
}
