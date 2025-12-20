package graphql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
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

	// accountNumber is the Octopus Energy Germany account number
	accountNumber string

	// tokenSource manages OAuth2 token lifecycle
	oauth2.TokenSource
}

// NewClient returns a new, authenticated instance of OctopusDeGraphQLClient.
func NewClient(log *util.Logger, email, password, accountNumber string) (*OctopusDeGraphQLClient, error) {
	gq := &OctopusDeGraphQLClient{
		log:           log,
		email:         email,
		password:      password,
		accountNumber: accountNumber,
	}

	// Initialize TokenSource with oauth pattern
	gq.TokenSource = oauth.RefreshTokenSource(new(oauth2.Token), gq)

	// Create HTTP client with OAuth2 transport
	cli := request.NewClient(log)
	cli.Transport = &oauth2.Transport{
		Base:   cli.Transport,
		Source: gq.TokenSource,
	}

	gq.Client = graphql.NewClient(BaseURI, cli)

	return gq, nil
}

// RefreshToken implements oauth.TokenRefresher to obtain a new JWT token.
// It parses the JWT to extract the actual expiry time from the token claims.
func (c *OctopusDeGraphQLClient) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create a temporary client without authentication for the token request
	cli := request.NewClient(c.log)
	tempClient := graphql.NewClient(BaseURI, cli)

	var q krakenTokenAuthentication
	if err := tempClient.Mutate(ctx, &q, map[string]any{
		"email":    c.email,
		"password": c.password,
	}); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Parse JWT to extract expiry time
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(q.ObtainKrakenToken.Token, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims format")
	}

	// Extract expiry from JWT claims
	var expiry time.Time
	if exp, ok := claims["exp"].(float64); ok {
		expiry = time.Unix(int64(exp), 0)
	} else {
		// Fallback to 1 hour if exp claim is missing
		c.log.DEBUG.Println("JWT exp claim missing, using 1 hour default")
		expiry = time.Now().Add(time.Hour)
	}

	c.log.TRACE.Printf("GraphQL: refreshed token, expires at %s", expiry)

	return &oauth2.Token{
		AccessToken: q.ObtainKrakenToken.Token,
		Expiry:      expiry,
	}, nil
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
