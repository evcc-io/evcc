// Package ngeso implements the carbonintensity.org.uk Grid CO2 tracking service, which provides CO2 forecasting for the UK at a national and regional level.
// This service is provided by the National Grid Electricity System Operator (NGESO).
package ngeso

import (
	"fmt"
	"time"
)

// BaseURI is the root path that the API is accessed from.
const BaseURI = "https://api.carbonintensity.org.uk/"

// ForecastNationalURI defines the location of the national forecast.
// Replace the first %s with the RFC3339 timestamp to fetch from.
const ForecastNationalURI = BaseURI + "intensity/%s/fw48h"

// ForecastRegionalByIdURI defines the location of the regional forecast determined by Region ID.
// Replace the first %s with the RFC3339 timestamp to fetch from, and the second with the appropriate Region ID.
const ForecastRegionalByIdURI = BaseURI + "regional/intensity/%s/fw48h/regionid/%s"

// ForecastRegionalByPostcodeURI defines the location of the regional forecast determined by a given postcode.
// Replace the first %s with the RFC3339 timestamp to fetch from, and the second with the appropriate postcode.
const ForecastRegionalByPostcodeURI = BaseURI + "regional/intensity/%s/fw48h/postcode/%s"

// ConstructForecastNationalAPI returns a validly formatted, fully qualified URI to the national forecast.
func ConstructForecastNationalAPI() string {
	currentTs := time.Now().UTC()
	t := currentTs.Format(time.RFC3339)
	return fmt.Sprintf(ForecastNationalURI, t)
}

// ConstructForecastRegionalByIdAPI returns a validly formatted, fully qualified URI to the forecast valid for the given region.
func ConstructForecastRegionalByIdAPI(r string) string {
	currentTs := time.Now().UTC()
	t := currentTs.Format(time.RFC3339)
	return fmt.Sprintf(ForecastRegionalByIdURI, t, r)
}

// ConstructForecastRegionalByPostcodeAPI returns a validly formatted, fully qualified URI to the forecast valid for the given postcode.
func ConstructForecastRegionalByPostcodeAPI(p string) string {
	currentTs := time.Now().UTC()
	t := currentTs.Format(time.RFC3339)
	return fmt.Sprintf(ForecastRegionalByPostcodeURI, t, p)
}

type RegionalIntensityResult struct {
	RegionId  int            `json:"regionid"`
	DNORegion string         `json:"dnoregion"`
	ShortName string         `json:"shortname"`
	Results   IntensityRates `json:"data"`
}
type IntensityRates struct {
	Results []CarbonIntensityForecastEntry `json:"data"`
}

type CarbonIntensityForecastEntry struct {
	ValidityStart shortRFC3339Timestamp `json:"from"`
	ValidityEnd   shortRFC3339Timestamp `json:"to"`
	Intensity     CarbonIntensity       `json:"intensity"`
}

type CarbonIntensity struct {
	// The forecasted rate in gCO2/kWh
	Forecast float64 `json:"forecast"`
	// The rate recorded when this slot occurred - only available historically, otherwise nil
	Actual float64 `json:"actual"`
	// A human-readable representation of the level of emissions (e.g "low", "moderate")
	Index string `json:"index"`
}
