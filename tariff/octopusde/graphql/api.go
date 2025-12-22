package graphql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
)

// BaseURI is Octopus Energy Germany's Kraken API root.
// The implementation in this file follows the published example at https://octopusenergy.de/blog/wohnen/dynamisch-sparen-per-api
const BaseURI = "https://api.oeg-kraken.energy/v1/graphql/"

// OctopusDeGraphQLClient provides an interface for communicating with Octopus Energy Germany's Kraken platform.
type OctopusDeGraphQLClient struct {
	*graphql.Client

	// Local logging utility.
	log *util.Logger

	// email is the Octopus Energy Germany account email
	email string

	// password is the Octopus Energy Germany account password
	password string

	// token is the GraphQL token used for communication with kraken
	token *string
	// tokenExpiration tracks the expiry of the acquired token
	tokenExpiration time.Time
	// tokenMtx should be held when requesting a new token
	tokenMtx sync.Mutex

	// accountNumber is the Octopus Energy Germany account number
	accountNumber string
}

// NewClient returns a new, authenticated instance of OctopusDeGraphQLClient.
func NewClient(log *util.Logger, email, password, accountNumber string) (*OctopusDeGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusDeGraphQLClient{
		Client:        graphql.NewClient(BaseURI, cli),
		log:           log,
		email:         email,
		password:      password,
		accountNumber: accountNumber,
	}

	if err := gq.refreshToken(); err != nil {
		return nil, err
	}

	// Future requests must have the appropriate Authorization header set
	gq.Client = gq.Client.WithRequestModifier(func(r *http.Request) {
		gq.tokenMtx.Lock()
		defer gq.tokenMtx.Unlock()
		if gq.token != nil {
			r.Header.Add("Authorization", *gq.token)
		}
	})

	return gq, nil
}

// refreshToken updates the GraphQL token from the email and password.
// Basic caching is provided - it will not update the token if it hasn't expired yet.
func (c *OctopusDeGraphQLClient) refreshToken() error {
	// take a lock against the token mutex for the refresh
	c.tokenMtx.Lock()
	defer c.tokenMtx.Unlock()

	if time.Until(c.tokenExpiration) > 5*time.Minute {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create a temporary client without authentication for the initial token request
	cli := request.NewClient(c.log)
	tempClient := graphql.NewClient(BaseURI, cli)

	var q krakenTokenAuthentication
	if err := tempClient.Mutate(ctx, &q, map[string]any{
		"email":    c.email,
		"password": c.password,
	}); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.token = &q.ObtainKrakenToken.Token
	c.tokenExpiration = time.Now().Add(time.Hour)
	c.log.TRACE.Println("GraphQL: refreshed token, now expires", c.tokenExpiration)
	return nil
}

// UnitRateForecast queries the day-ahead price forecast for the account
func (c *OctopusDeGraphQLClient) UnitRateForecast() ([]RatePeriod, error) {
	// Update refresh token (if necessary)
	if err := c.refreshToken(); err != nil {
		return nil, err
	}

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
					c.log.DEBUG.Printf("failed to parse net unit rate '%s': %v", rate.NetUnitRateCentsPerKwh, err)
					return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
				}

				grossRate, err := parseFloat(rate.LatestGrossUnitRateCentsPerKwh)
				if err != nil {
					c.log.DEBUG.Printf("failed to parse gross unit rate '%s': %v", rate.LatestGrossUnitRateCentsPerKwh, err)
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

	c.log.TRACE.Printf("GraphQL: retrieved %d rate periods", len(rates))
	return rates, nil
}

// parseFloat parses a string to float64, handling the specific format used by Octopus API
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
