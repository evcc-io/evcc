// Package entsoe implements a minimalized version of the European Network of Transmission System Operators for Electricity's
// Transparency Platform API (https://transparency.entsoe.eu)
package entsoe

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
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

// RateData defines the per-unit Value over a period of time spanning ValidityStart and ValidityEnd.
type RateData struct {
	ValidityStart time.Time
	ValidityEnd   time.Time
	Value         float64
}

// GetTsPriceData accepts a set of TimeSeries data entries, and
// returns a sorted array of RateData based on the timestamp of each data entry.
func GetTsPriceData(ts []TimeSeries, resolution ResolutionType) ([]RateData, error) {
	for _, v := range ts {
		if v.Period.Resolution != resolution {
			continue
		}

		data, err := ExtractTsPriceData(&v)
		if err != nil {
			return nil, err
		}

		// Now sort all entries by timestamp.
		// Not sure if this is entirely necessary for evcc's use, could consider removing this if it becomes a performance issue.
		sort.Slice(data, func(i, j int) bool {
			return data[i].ValidityStart.Before(data[j].ValidityStart)
		})

		return data, nil
	}

	return nil, fmt.Errorf("no data for resolution: %v", resolution)
}

// ExtractTsPriceData massages the given TimeSeries data set to provide RateData entries with associated start and end timestamps.
func ExtractTsPriceData(timeseries *TimeSeries) ([]RateData, error) {
	data := make([]RateData, 0, len(timeseries.Period.Point))

	duration, err := iso8601.ParseDuration(string(timeseries.Period.Resolution))
	if err != nil {
		return nil, err
	}

	// tCurrencyUnit := timeseries.CurrencyUnitName
	// tPriceMeasureUnit := timeseries.PriceMeasureUnitName
	// Brief check just to make sure we're about to decode the data as expected.
	if timeseries.PriceMeasureUnitName != "MWH" {
		return nil, fmt.Errorf("%w: price data not in expected unit", ErrInvalidData)
	}

	tPointer := timeseries.Period.TimeInterval.Start.Time
	for _, point := range timeseries.Period.Point {
		d := RateData{
			Value:         point.PriceAmount / 1e3, // Price/MWh to Price/kWh
			ValidityStart: tPointer,
		}

		// Nudge pointer on as required by defined data resolution
		switch timeseries.Period.Resolution {
		case ResolutionQuarterHour, ResolutionHalfHour, ResolutionHour:
			tPointer = tPointer.Add(duration)
		case ResolutionDay:
			tPointer = tPointer.AddDate(0, 0, 1)
		case ResolutionWeek:
			tPointer = tPointer.AddDate(0, 0, 7)
		case ResolutionYear:
			tPointer = tPointer.AddDate(0, 1, 0)
		default:
			return nil, fmt.Errorf("invalid resolution: %v", timeseries.Period.Resolution)
		}
		d.ValidityEnd = tPointer

		data = append(data, d)
	}

	return data, nil
}
