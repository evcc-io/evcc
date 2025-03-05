package octopusde

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
)

// Implementation of intelligent dispatch times is WIP //
// krakenGraphQLURL_DE is the GraphQL query endpoint for Octopus Energy Germany.
const (
	krakenGraphQLURL_DE = "https://api.oeg-kraken.energy/v1/graphql/"
)

// OctopusDEGraphQLClient provides an interface for communicating with Octopus Energy's Kraken platform.
type OctopusDEGraphQLClient struct {
	*graphql.Client
	httpClient *http.Client

	// email and password are the Octopus Energy account credentials (provided by user)
	email    string
	password string

	// token is the GraphQL token used for communication with kraken (we get this ourselves with the email and password)
	token *string

	// tokenExpiration tracks the expiry of the acquired token. A new Token should be obtained if this time is passed.
	tokenExpiration time.Time

	// tokenMtx should be held when requesting a new token.
	tokenMtx sync.Mutex

	// accountNumber is the Octopus Energy account number associated with the given credentials (queried ourselves via GraphQL)
	accountNumber string

	// timeout is the duration for which the client will wait for a response from the server.
	timeout time.Duration

	// log is the logger used for logging messages.
	log *util.Logger

	// subscriptionClient is the GraphQL subscription client.
	subscriptionClient *graphql.SubscriptionClient
}

type headerRoundTripper struct {
	Transport http.RoundTripper
	headers   map[string]string
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range h.headers {
		req.Header.Add(key, value)
	}
	return h.Transport.RoundTrip(req)
}

// NewClient returns a new, unauthenticated instance of OctopusGraphQLClient.
func NewClient(log *util.Logger, email, password string) (*OctopusDEGraphQLClient, error) {
	cli := request.NewClient(log)

	subscriptionClient := graphql.NewSubscriptionClient(krakenGraphQLURL_DE).
		WithProtocol(graphql.GraphQLWS).
		WithWebSocketOptions(graphql.WebsocketOptions{
			HTTPClient: &http.Client{
				Transport: headerRoundTripper{
					Transport: http.DefaultTransport,
					headers: map[string]string{
						"Authorization": "",
					},
				},
			},
		}).
		WithLog(log.TRACE.Println)

	gq := &OctopusDEGraphQLClient{
		Client:             graphql.NewClient(krakenGraphQLURL_DE, cli),
		httpClient:         cli,
		email:              email,
		password:           password,
		timeout:            10 * time.Second,
		log:                log,
		subscriptionClient: subscriptionClient,
	}

	if err := gq.RefreshToken(); err != nil {
		return nil, err
	}

	// Future requests must have the appropriate Authorization header set.
	gq.Client = gq.Client.WithRequestModifier(func(r *http.Request) {
		gq.tokenMtx.Lock()
		defer gq.tokenMtx.Unlock()
		r.Header.Add("Authorization", *gq.token)
	})

	// Set the Authorization header for the subscription client
	subscriptionClient = subscriptionClient.WithWebSocketOptions(graphql.WebsocketOptions{
		HTTPClient: &http.Client{
			Transport: headerRoundTripper{
				Transport: http.DefaultTransport,
				headers: map[string]string{
					"Authorization": *gq.token,
				},
			},
		},
	}).
		WithLog(log.TRACE.Println)

	return gq, nil
}

// RefreshToken updates the GraphQL token from the set email and password.
// Basic caching is provided - it will not update the token if it hasn't expired yet.
func (c *OctopusDEGraphQLClient) RefreshToken() error {
	// take a lock against the token mutex for the refresh
	c.tokenMtx.Lock()
	defer c.tokenMtx.Unlock()

	if time.Until(c.tokenExpiration) > 5*time.Minute {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var q krakenDETokenAuthentication
	if err := c.Client.Mutate(ctx, &q, map[string]interface{}{"email": c.email, "password": c.password}); err != nil {
		return err
	}

	c.token = &q.ObtainKrakenToken.Token
	c.tokenExpiration = time.Now().Add(time.Hour)
	return nil
}

// AccountNumber queries the Account Number assigned to the associated credentials.
// Caching is provided.
func (c *OctopusDEGraphQLClient) AccountNumber() (string, error) {
	// Check cache
	if c.accountNumber != "" {
		return c.accountNumber, nil
	}

	// Update refresh token (if necessary)
	if err := c.RefreshToken(); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var q krakenDEAccountLookup
	if err := c.Client.Query(ctx, &q, nil); err != nil {
		return "", err
	}

	if len(q.Viewer.Accounts) == 0 {
		return "", errors.New("no account associated with given octopus credentials")
	}
	if len(q.Viewer.Accounts) > 1 {
		return "", errors.New("more than one octopus account on this email not supported")
	}
	c.accountNumber = q.Viewer.Accounts[0].Number
	return c.accountNumber, nil
}

// Rate represents a single price rate with its time slot
type Rate struct {
	Price     float64
	StartTime string
	EndTime   string
	Name      string
}

// QueryWithMap executes a raw GraphQL query with a map[string]interface{} result
func (c *OctopusDEGraphQLClient) QueryWithMap(ctx context.Context, result *map[string]interface{}, query string, variables map[string]interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, krakenGraphQLURL_DE, nil)
	if err != nil {
		return err
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set authorization header
	c.tokenMtx.Lock()
	req.Header.Set("Authorization", *c.token)
	c.tokenMtx.Unlock()

	// Construct the request body
	type graphQLReq struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables,omitempty"`
	}

	body := graphQLReq{
		Query:     query,
		Variables: variables,
	}

	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
		return fmt.Errorf("error encoding GraphQL request: %w", err)
	}

	req.Body = io.NopCloser(&requestBody)
	req.ContentLength = int64(requestBody.Len())

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse the response
	var responseBody struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return fmt.Errorf("error decoding GraphQL response: %w", err)
	}

	// Check for GraphQL errors
	if len(responseBody.Errors) > 0 {
		errorMessages := make([]string, len(responseBody.Errors))
		for i, err := range responseBody.Errors {
			errorMessages[i] = err.Message
		}
		return fmt.Errorf("GraphQL errors: %v", errorMessages)
	}

	// Copy data to result
	*result = responseBody.Data
	return nil
}

// FetchRates fetches the detailed rate information for the given account number.
func (c *OctopusDEGraphQLClient) FetchRates(accountNumber string) ([]Rate, error) {
	// Try to fetch using the detailed query first
	rates, err := c.fetchTimeOfUseRates(accountNumber)
	if err != nil {
		c.log.WARN.Printf("Failed to fetch time-of-use rates: %v. Falling back to simple rates.", err)
		// If detailed query fails, fall back to the simpler gross rate query
		return c.fetchSimpleRates(accountNumber)
	}
	return rates, nil
}

// fetchTimeOfUseRates attempts to fetch the detailed time-of-use rates
func (c *OctopusDEGraphQLClient) fetchTimeOfUseRates(accountNumber string) ([]Rate, error) {
	// Define the GraphQL query directly
	query := `
	query AccountRates($accountNumber: String!) {
		account(accountNumber: $accountNumber) {
			allProperties {
				electricityMalos {
					agreements {
						unitRateInformation {
							__typename
							... on SimpleProductUnitRateInformation {
								grossRateInformation {
									date
									grossRate
									rateValidToDate
									vatRate
								}
								latestGrossUnitRateCentsPerKwh
								netUnitRateCentsPerKwh
							}
							... on TimeOfUseProductUnitRateInformation {
								rates {
									grossRateInformation {
										date
										grossRate
										rateValidToDate
										vatRate
									}
									latestGrossUnitRateCentsPerKwh
									netUnitRateCentsPerKwh
									timeslotActivationRules {
										activeFromTime
										activeToTime
									}
									timeslotName
								}
							}
						}
					}
				}
			}
		}
	}`

	c.log.TRACE.Printf("Starting GraphQL query for account rates: %s", accountNumber)

	// Execute the query with backoff retry
	var result map[string]interface{}
	if err := backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()

		variables := map[string]interface{}{
			"accountNumber": accountNumber,
		}

		// Use our custom QueryWithMap method
		err := c.QueryWithMap(ctx, &result, query, variables)
		if err != nil {
			c.log.ERROR.Printf("GraphQL query error: %v", err)
			return err
		}

		c.log.TRACE.Printf("GraphQL query successful")
		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)); err != nil {
		return nil, fmt.Errorf("failed to fetch rates: %w", err)
	}

	// Process the rates from the response safely
	var rates []Rate

	// Safe traversal of the JSON response
	account, ok := result["account"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing account in response")
	}

	properties, ok := account["allProperties"].([]interface{})
	if !ok || len(properties) == 0 {
		return nil, errors.New("no properties in response")
	}

	property := properties[0].(map[string]interface{})
	electricityMalos, ok := property["electricityMalos"].([]interface{})
	if !ok || len(electricityMalos) == 0 {
		return nil, errors.New("no electricity meters in response")
	}

	malos := electricityMalos[0].(map[string]interface{})
	agreements, ok := malos["agreements"].([]interface{})
	if !ok || len(agreements) == 0 {
		return nil, errors.New("no agreements in response")
	}

	// Process the first agreement (could handle multiple in the future)
	agreement := agreements[0].(map[string]interface{})
	rateInfo, ok := agreement["unitRateInformation"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing unit rate information")
	}

	typename, _ := rateInfo["__typename"].(string)
	c.log.TRACE.Printf("Rate type: %s", typename)

	switch typename {
	case "TimeOfUseProductUnitRateInformation":
		// Handle TOU rates
		ratesArray, ok := rateInfo["rates"].([]interface{})
		if ok && len(ratesArray) > 0 {
			for _, r := range ratesArray {
				rate, ok := r.(map[string]interface{})
				if !ok {
					continue
				}

				// Get rate value
				rateValue, ok := rate["latestGrossUnitRateCentsPerKwh"].(string)
				if !ok {
					continue
				}

				rateFloat, err := strconv.ParseFloat(rateValue, 64)
				if err != nil {
					c.log.WARN.Printf("Failed to parse rate: %v", err)
					continue
				}

				// Convert from cents to euros
				rateEuros := rateFloat / 100.0

				timeslotName, _ := rate["timeslotName"].(string)

				// Get time slots
				timeSlots, ok := rate["timeslotActivationRules"].([]interface{})
				if !ok || len(timeSlots) == 0 {
					// If no time slots are defined, assume all day
					rates = append(rates, Rate{
						Price:     rateEuros,
						StartTime: "00:00:00",
						EndTime:   "00:00:00",
						Name:      timeslotName,
					})
					c.log.TRACE.Printf("Added TOU rate (no slots): %.4f €/kWh (%s)", rateEuros, timeslotName)
					continue
				}

				// Process each time slot
				for _, ts := range timeSlots {
					timeSlot, ok := ts.(map[string]interface{})
					if !ok {
						continue
					}

					startTime, _ := timeSlot["activeFromTime"].(string)
					endTime, _ := timeSlot["activeToTime"].(string)

					if startTime == "" {
						startTime = "00:00:00"
					}

					if endTime == "" {
						endTime = "00:00:00"
					}

					rates = append(rates, Rate{
						Price:     rateEuros,
						StartTime: startTime,
						EndTime:   endTime,
						Name:      timeslotName,
					})

					c.log.TRACE.Printf("Added TOU rate: %.4f €/kWh from %s to %s (%s)",
						rateEuros, startTime, endTime, timeslotName)
				}
			}
		}

	case "SimpleProductUnitRateInformation":
		// Handle simple product rate
		rateValue, ok := rateInfo["latestGrossUnitRateCentsPerKwh"].(string)
		if ok {
			rateFloat, err := strconv.ParseFloat(rateValue, 64)
			if err == nil {
				rateEuros := rateFloat / 100.0
				rates = append(rates, Rate{
					Price:     rateEuros,
					StartTime: "00:00:00",
					EndTime:   "00:00:00",
					Name:      "STANDARD",
				})
				c.log.TRACE.Printf("Added simple rate: %.4f €/kWh (all day)", rateEuros)
			}
		}
	}

	if len(rates) == 0 {
		return nil, errors.New("no valid rates found in response")
	}

	return rates, nil
}

// fetchSimpleRates is a fallback method to fetch just the gross rates
func (c *OctopusDEGraphQLClient) fetchSimpleRates(accountNumber string) ([]Rate, error) {
	client := graphql.NewClient(krakenGraphQLURL_DE, http.DefaultClient).
		WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", *c.token)
		})

	var query krakenDEAccountGrossRate

	c.log.TRACE.Printf("Starting simple GraphQL query for account number: %s", accountNumber)

	if err := backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()
		variables := map[string]interface{}{
			"accountNumber": graphql.String(accountNumber),
		}
		err := client.Query(ctx, &query, variables)
		if err != nil {
			c.log.ERROR.Printf("GraphQL query error: %v", err)
			return err
		}
		c.log.TRACE.Printf("GraphQL query successful")
		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)); err != nil {
		// If all attempts fail, use a default rate
		c.log.ERROR.Printf("Failed to fetch rates after retries, using default: %v", err)
		return []Rate{
			{
				Price:     0.2827, // Default fallback rate
				StartTime: "00:00:00",
				EndTime:   "00:00:00",
				Name:      "DEFAULT",
			},
		}, nil
	}

	// Convert grossRate from string to float64 and then from cents to euros
	var grossRate float64 = 0.2827 // Default fallback

	// Safely navigate the response
	if len(query.Account.AllProperties) > 0 &&
		len(query.Account.AllProperties[0].ElectricityMalos) > 0 &&
		len(query.Account.AllProperties[0].ElectricityMalos[0].Agreements) > 0 &&
		len(query.Account.AllProperties[0].ElectricityMalos[0].Agreements[0].UnitRateGrossRateInformation) > 0 {

		rateStr := query.Account.AllProperties[0].ElectricityMalos[0].Agreements[0].UnitRateGrossRateInformation[0].GrossRate
		if rateStr != "" {
			rate, err := strconv.ParseFloat(rateStr, 64)
			if err == nil {
				grossRate = rate / 100
				c.log.TRACE.Printf("Found gross rate: %f", grossRate)
			} else {
				c.log.ERROR.Printf("Error converting gross rate: %v", err)
			}
		}
	} else {
		c.log.WARN.Printf("Incomplete rate information in response, using default rate")
	}

	return []Rate{
		{
			Price:     grossRate,
			StartTime: "00:00:00",
			EndTime:   "00:00:00",
			Name:      "STANDARD",
		},
	}, nil
}

// FetchGrossRates is maintained for backwards compatibility
func (c *OctopusDEGraphQLClient) FetchGrossRates(accountNumber string) ([]float64, error) {
	rates, err := c.FetchRates(accountNumber)
	if err != nil {
		return []float64{0.2827}, nil // Return default rate on error
	}

	// Just return the first rate price for backwards compatibility
	if len(rates) > 0 {
		return []float64{rates[0].Price}, nil
	}

	return []float64{0.2827}, nil // Default fallback rate
}
