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

	for _, v := range ts {
		if v.Period.Resolution != resolution {
			continue
		}

		data, err := ExtractTsPriceData(&v)
		if err != nil {
			return nil, err
		}

		res = append(res, data...)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("no data for resolution: %v", resolution)
	}

	return res, nil
}

// ExtractTsPriceData massages the given TimeSeries data set to provide Rate entries with associated start and end timestamps.
func ExtractTsPriceData(timeseries *TimeSeries) ([]Rate, error) {
	data := make([]Rate, 0, len(timeseries.Period.Point))

	duration, err := iso8601.ParseDuration(string(timeseries.Period.Resolution))
	if err != nil {
		return nil, err
	}

	if unit := timeseries.PriceMeasureUnitName; unit != "MWH" {
		return nil, fmt.Errorf("%w: invalid unit: %s", ErrInvalidData, unit)
	}

	ts := timeseries.Period.TimeInterval.Start.Time
	for _, point := range timeseries.Period.Point {
		d := Rate{
			Value: point.PriceAmount / 1e3, // Price/MWh to Price/kWh
			Start: ts,
		}

		// Nudge pointer on as required by defined data resolution
		switch timeseries.Period.Resolution {
		case ResolutionQuarterHour, ResolutionHalfHour, ResolutionHour:
			ts = ts.Add(duration)
		case ResolutionDay:
			ts = ts.AddDate(0, 0, 1)
		case ResolutionWeek:
			ts = ts.AddDate(0, 0, 7)
		case ResolutionYear:
			ts = ts.AddDate(1, 0, 0)
		default:
			return nil, fmt.Errorf("%w: invalid resolution: %v", ErrInvalidData, timeseries.Period.Resolution)
		}
		d.End = ts

		data = append(data, d)
	}

	return data, nil
}
