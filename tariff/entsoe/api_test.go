package entsoe

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/shortrfc3339"
)

func hourlySeries(position int, price float64) TimeSeries {
	start := time.Date(2026, 7, 1, 22, 0, 0, 0, time.UTC)

	ts := TimeSeries{
		PriceMeasureUnitName: "MWH",
		ClassificationSequenceAttributeInstanceComponentPosition: position,
	}

	period := TimeSeriesPeriod{Resolution: ResolutionHour}
	period.TimeInterval.Start = shortrfc3339.Timestamp{Time: start}
	period.TimeInterval.End = shortrfc3339.Timestamp{Time: start.Add(24 * time.Hour)}

	for i := 1; i <= 24; i++ {
		period.Point = append(period.Point, Point{Position: i, PriceAmount: price})
	}

	ts.Period = []TimeSeriesPeriod{period}

	return ts
}

// A single TimeSeries at position 2 is still valid data and must not be discarded.
func TestGetTsPriceDataSinglePosition2(t *testing.T) {
	res, err := GetTsPriceData([]TimeSeries{hourlySeries(2, 100)}, ResolutionHour)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 24 {
		t.Fatalf("expected 24 rates, got %d", len(res))
	}
}

// When two TimeSeries cover the same interval, the lower classification position wins.
func TestGetTsPriceDataDualPositionSameInterval(t *testing.T) {
	ts := []TimeSeries{
		hourlySeries(2, 999),
		hourlySeries(1, 100),
	}

	res, err := GetTsPriceData(ts, ResolutionHour)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range res {
		if r.Value != 100.0/1e3 {
			t.Fatalf("expected position 1 data (100), got %v", r.Value*1e3)
		}
	}
}
