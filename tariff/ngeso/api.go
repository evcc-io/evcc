// Package ngeso implements the carbonintensity.org.uk Grid CO2 tracking service, which provides CO2 forecasting for the UK at a national and regional level.
// This service is provided by the National Grid Electricity System Operator (NGESO).
package ngeso

import (
	"errors"
	"fmt"
	"github.com/evcc-io/evcc/util/request"
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

// ConstructNationalForecastRequest returns a request object to be used when calling the national API.
func ConstructNationalForecastRequest() *CarbonForecastNationalRequest {
	return &CarbonForecastNationalRequest{}
}

// ConstructRegionalForecastByIDRequest returns a validly formatted, fully qualified URI to the forecast valid for the given region.
func ConstructRegionalForecastByIDRequest(r string) *CarbonForecastRegionalRequest {
	return &CarbonForecastRegionalRequest{regionid: r}
}

// ConstructRegionalForecastByPostcodeRequest returns a validly formatted, fully qualified URI to the forecast valid for the given postcode.
func ConstructRegionalForecastByPostcodeRequest(p string) *CarbonForecastRegionalRequest {
	return &CarbonForecastRegionalRequest{postcode: p}
}

type CarbonForecastRequest interface {
	URI() (string, error)
	DoRequest(helper *request.Helper) (CarbonForecastResponse, error)
}

type CarbonForecastNationalRequest struct{}

func (r *CarbonForecastNationalRequest) URI() (string, error) {
	return fmt.Sprintf(ForecastNationalURI, time.Now().UTC().Format(time.RFC3339)), nil
}

func (r *CarbonForecastNationalRequest) DoRequest(client *request.Helper) (CarbonForecastResponse, error) {
	uri, err := r.URI()
	if err != nil {
		return nil, err
	}
	var res NationalIntensityResult
	err = client.GetJSON(uri, res)
	return res, err
}

type CarbonForecastRegionalRequest struct {
	regionid string
	postcode string
}

func (r *CarbonForecastRegionalRequest) URI() (string, error) {
	currentTs := time.Now().UTC().Format(time.RFC3339)
	// Prefer postcode to Region ID
	if r.postcode != "" {
		return fmt.Sprintf(ForecastRegionalByPostcodeURI, currentTs, r.postcode), nil
	}
	if r.regionid != "" {
		return fmt.Sprintf(ForecastRegionalByIdURI, currentTs, r.regionid), nil
	}

	// One of the region identifiers must be supplied, if neither are then just return an error
	return "", ErrRegionalRequestInvalidFormat
}

func (r *CarbonForecastRegionalRequest) DoRequest(client *request.Helper) (CarbonForecastResponse, error) {
	uri, err := r.URI()
	if err != nil {
		return nil, err
	}
	res := &RegionalIntensityResult{}
	err = client.GetJSON(uri, &res)
	return res, err
}

type CarbonForecastResponse interface {
	Results() []CarbonIntensityForecastEntry
}

// RegionalIntensityResult is returned by Regional requests. It wraps all data inside a data element.
// Because that makes sense, and makes all of this SO much easier. /s
type RegionalIntensityResult struct {
	Data RegionalIntensityResultData `json:"data"`
}

func (r RegionalIntensityResult) Results() []CarbonIntensityForecastEntry {
	return r.Data.Rates
}

// RegionalIntensityResultData is returned by Regional requests. It includes a bit of extra data.
type RegionalIntensityResultData struct {
	RegionId  int                            `json:"regionid"`
	DNORegion string                         `json:"dnoregion"`
	ShortName string                         `json:"shortname"`
	Rates     []CarbonIntensityForecastEntry `json:"data"`
}

// NationalIntensityResult is returned either as a sub-element of a Regional request, or as the main result of a National request.
type NationalIntensityResult struct {
	Rates []CarbonIntensityForecastEntry `json:"data"`
}

// Results is a helper / interface function to return the current rate data.
func (r NationalIntensityResult) Results() []CarbonIntensityForecastEntry {
	return r.Rates
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

var ErrRegionalRequestInvalidFormat = errors.New("regional request object missing region")
