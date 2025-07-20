// Package entsoe implements a minimalized version of the European Network of Transmission System Operators for Electricity's
// Transparency Platform API (https://transparency.entsoe.eu)
package entsoe

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/dylanmei/iso8601"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
)

const (
	// BaseURI is the root path that the API is accessed from.
	BaseURI = "https://web-api.tp.entsoe.eu/api"

	// numericDateFormat is a time.Parse compliant formatting string for the numeric date format used by entsoe get requests.
	numericDateFormat = "200601021504"
)

var ErrInvalidData = errors.New("invalid data received")

// DayAheadPricesRequest constructs a new DayAheadPricesRequest.
func DayAheadPricesRequest(domain string, duration time.Duration) *http.Request {
	now := time.Now().Truncate(time.Hour)

	params := url.Values{
		"DocumentType": {string(ProcessTypeDayAhead)},
		"In_Domain":    {domain},
		"Out_Domain":   {domain},
		"PeriodStart":  {now.Format(numericDateFormat)},
		"PeriodEnd":    {now.Add(duration).Format(numericDateFormat)},
	}

	uri := BaseURI + "?" + params.Encode()
	req, _ := request.New(http.MethodGet, uri, nil, request.AcceptXML)

	return req
}

// Rate defines the per-unit Value over a period of time spanning Start and End.
type Rate struct {
	Start time.Time
	End   time.Time
	Value float64
}

// GetTsPriceData accepts a set of TimeSeries data entries, and
// returns a sorted array of Rate based on the timestamp of each data entry.
func GetTsPriceData(ts []TimeSeries, resolution ResolutionType) ([]Rate, error) {
	var res []Rate

	for _, ts := range ts {
		if unit := ts.PriceMeasureUnitName; unit != "MWH" {
			return nil, fmt.Errorf("%w: invalid unit: %s", ErrInvalidData, unit)
		}

		for _, period := range ts.Period {
			if period.Resolution != resolution {
				continue
			}

			data, err := ExtractPeriodPriceData(&period)
			if err != nil {
				return nil, err
			}

			res = append(res, data...)
		}
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("no data for resolution: %v", resolution)
	}

	return res, nil
}

// ExtractPeriodPriceData massages the given Period data set to provide Rate entries with associated start and end timestamps.
func ExtractPeriodPriceData(period *TimeSeriesPeriod) ([]Rate, error) {
	data := make([]Rate, 0, len(period.Point))

	duration, err := iso8601.ParseDuration(string(period.Resolution))
	if err != nil {
		return nil, err
	}

	var count int
	switch period.Resolution {
	case ResolutionHour:
		count = 24
	case ResolutionHalfHour:
		count = 2 * 24
	case ResolutionQuarterHour:
		count = 4 * 24
	default:
		return nil, fmt.Errorf("%w: invalid resolution: %v", ErrInvalidData, period.Resolution)
	}

	ts := period.TimeInterval.Start.Time
	points := lo.SliceToMap(period.Point, func(p Point) (int, Point) {
		return p.Position, p
	})

	for pos := 1; pos <= count; pos++ {
		var point Point
		for last := pos; last > 0; last-- {
			if p, ok := points[last]; ok {
				point = p
				break
			}
		}

		if point.Position == 0 {
			return nil, fmt.Errorf("%w: missing point at position: %d", ErrInvalidData, pos)
		}

		start := ts.Add(time.Duration(pos-1) * duration)

		d := Rate{
			Value: point.PriceAmount / 1e3, // Price/MWh to Price/kWh
			Start: start,
			End:   start.Add(duration),
		}

		data = append(data, d)
	}

	return data, nil
}
