// Package entsoe implements a minimalized version of the European Network of Transmission System Operators for Electricity's
// Transparency Platform API (https://transparency.entsoe.eu)
package entsoe

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/dylanmei/iso8601"
	"github.com/evcc-io/evcc/util/request"
)

// BaseURI is the root path that the API is accessed from.
const BaseURI = "https://web-api.tp.entsoe.eu/api"

// numericDateFormat is a time.Parse compliant formatting string for the numeric date format used by entsoe get requests.
const numericDateFormat = "200601021504"

var ErrInvalidData = errors.New("invalid data received")

// DayAheadPricesRequest represents a helper struct for requesting 4.2.10 Day Ahead Prices [12.1.D]
type DayAheadPricesRequest struct {
	domain      DomainType
	periodStart time.Time
	periodEnd   time.Time
}

// ConstructDayAheadPricesRequest constructs a new DayAheadPricesRequest.
// Domain and duration validity is not checked, that's on you.
func ConstructDayAheadPricesRequest(domain DomainType, duration time.Duration) DayAheadPricesRequest {
	// Round to the hour.
	now := time.Now().Truncate(time.Hour)
	return DayAheadPricesRequest{
		domain:      domain,
		periodStart: now,
		periodEnd:   now.Add(duration),
	}
}

// DoRequest requests the current Day Ahead Prices from ENTSOE.
// The client is expected to provide a decorator to provide the security key.
func (r *DayAheadPricesRequest) DoRequest(client *request.Helper) (PublicationMarketDocument, error) {
	var res PublicationMarketDocument

	// Currently opting to use GET request to keep it relatively simple, but POST is an option.
	// Would have to figure out building the XML requests in a sane, structured way though.

	params := url.Values{
		"DocumentType": {string(ProcessTypeDayAhead)},
		"In_Domain":    {r.domain},
		"Out_Domain":   {r.domain},
		"PeriodStart":  {r.periodStart.Format(numericDateFormat)},
		"PeriodEnd":    {r.periodEnd.Format(numericDateFormat)},
	}

	// Feels like we might be duplicating the wheel here, but this is nice and simple (and fast)
	uri := fmt.Sprintf("%s?%s", BaseURI, params.Encode())

	err := client.GetXML(uri, &res)
	return res, err
}

// RateData defines the per-unit Value over a period of time spanning ValidityStart and ValidityEnd.
type RateData struct {
	ValidityStart time.Time
	ValidityEnd   time.Time
	Value         float64
}

// GetTsPriceData accepts a set of TimeSeries data entries, and
// returns a sorted array of RateData based on the timestamp of each data entry.
func GetTsPriceData(ts *[]TimeSeries) (data []RateData, err error) {
	for _, v := range *ts {
		tsData, err := ExtractTsPriceData(&v)
		if err != nil {
			return data, err
		}
		// Just append the array for the time being, sort comes later.
		data = append(data, tsData...)
	}

	// Now sort all entries by timestamp.
	// Not sure if this is entirely necessary for evcc's use, could consider removing this if it becomes a performance issue.
	sort.Slice(data, func(i, j int) bool {
		return data[i].ValidityStart.Unix() < data[j].ValidityStart.Unix()
	})
	return data, nil
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
			Value:         point.PriceAmount / 100, // Price/MWH to Price/kWH
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
