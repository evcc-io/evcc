package graphql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

// BaseURI is Octopus Energy Germany's Kraken API root.
// The implementation in this file follows the published example at https://octopusenergy.de/blog/wohnen/dynamisch-sparen-per-api
const BaseURI = "https://api.oeg-kraken.energy/v1/graphql/"

// OctopusDeGraphQLClient provides an interface for communicating with Octopus Energy Germany's Kraken platform.
type OctopusDeGraphQLClient struct {
	log *util.Logger
	*graphql.Client
	accountNumber string
}

// NewClient returns a new, authenticated instance of OctopusDeGraphQLClient.
func NewClient(log *util.Logger, email, password, accountNumber string) (*OctopusDeGraphQLClient, error) {
	ts := oauth2.ReuseTokenSource(nil, &tokenSource{
		log:      log,
		email:    email,
		password: password,
	})

	cli := request.NewClient(log)
	cli.Transport = &transport.Decorator{
		Decorator: func(req *http.Request) error {
			token, err := ts.Token()
			if err != nil {
				return err
			}
			// Kraken API requires Authorization header without "Bearer" prefix
			req.Header.Set("Authorization", token.AccessToken)
			return nil
		},
		Base: cli.Transport,
	}

	gq := &OctopusDeGraphQLClient{
		log:           log,
		accountNumber: accountNumber,
		Client:        graphql.NewClient(BaseURI, cli),
	}

	return gq, nil
}

// UnitRateForecast queries the day-ahead price forecast for the account
func (c *OctopusDeGraphQLClient) UnitRateForecast() ([]RatePeriod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var q getDayAheadPrices
	if err := c.Client.Query(ctx, &q, map[string]any{
		"accountNumber": c.accountNumber,
	}); err != nil {
		return nil, err
	}

	// Extract rates from the query result
	if len(q.Account.Properties) == 0 {
		return nil, errors.New("no properties found")
	}

	// Find the active agreement across all properties
	var unitRateForecast []unitRateForecast
	for _, property := range q.Account.Properties {
		for _, malo := range property.ElectricityMalos {
			for _, agreement := range malo.Agreements {
				if agreement.IsActive {
					unitRateForecast = agreement.UnitRateForecast
					break
				}
			}
			if unitRateForecast != nil {
				break
			}
		}
		if unitRateForecast != nil {
			break
		}
	}

	if unitRateForecast == nil {
		return nil, errors.New("no active agreement found")
	}

	// Convert to RatePeriod slice
	var rates []RatePeriod
	for _, forecast := range unitRateForecast {
		// Extract the rate from the union type
		if forecast.UnitRateInformation.TimeOfUseProductUnitRateInformation.Rates != nil {
			for _, rate := range forecast.UnitRateInformation.TimeOfUseProductUnitRateInformation.Rates {
				// Parse string values to float64
				netRate, err := parseFloat(rate.NetUnitRateCentsPerKwh)
				if err != nil {
					return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
				}

				grossRate, err := parseFloat(rate.LatestGrossUnitRateCentsPerKwh)
				if err != nil {
					return nil, fmt.Errorf("failed to parse gross unit rate: %w", err)
				}

				rates = append(rates, RatePeriod{
					ValidFrom:                      forecast.ValidFrom,
					ValidTo:                        forecast.ValidTo,
					LatestGrossUnitRateCentsPerKwh: grossRate,
					NetUnitRateCentsPerKwh:         netRate,
				})
			}
		}
	}

	if len(rates) == 0 {
		return nil, errors.New("no rate forecast available")
	}

	return rates, nil
}

// parseFloat parses a string to float64, handling the specific format used by Octopus API
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
