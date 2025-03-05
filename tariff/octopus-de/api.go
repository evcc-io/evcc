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

const krakenGraphQLURL_DE = "https://api.oeg-kraken.energy/v1/graphql/"
const defaultRate = 0.2827 // Default fallback rate in EUR/kWh

// Rate represents a single price rate with its time slot
type Rate struct {
	Price     float64
	StartTime string
	EndTime   string
	Name      string
}

// OctopusDEGraphQLClient provides an interface for communicating with Octopus Energy's Kraken platform
type OctopusDEGraphQLClient struct {
	*graphql.Client
	httpClient      *http.Client
	email           string
	password        string
	token           *string
	tokenExpiration time.Time
	tokenMtx        sync.Mutex
	accountNumber   string
	timeout         time.Duration
	log             *util.Logger
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

// NewClient creates a new client for communicating with Octopus Energy
func NewClient(log *util.Logger, email, password string) (*OctopusDEGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusDEGraphQLClient{
		Client:     graphql.NewClient(krakenGraphQLURL_DE, cli),
		httpClient: cli,
		email:      email,
		password:   password,
		timeout:    10 * time.Second,
		log:        log,
	}

	if err := gq.RefreshToken(); err != nil {
		return nil, err
	}

	gq.Client = gq.Client.WithRequestModifier(func(r *http.Request) {
		gq.tokenMtx.Lock()
		defer gq.tokenMtx.Unlock()
		r.Header.Add("Authorization", *gq.token)
	})

	return gq, nil
}

// RefreshToken updates the GraphQL token
func (c *OctopusDEGraphQLClient) RefreshToken() error {
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

// AccountNumber queries the Account Number assigned to credentials
func (c *OctopusDEGraphQLClient) AccountNumber() (string, error) {
	if c.accountNumber != "" {
		return c.accountNumber, nil
	}

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

// QueryWithMap executes a raw GraphQL query
func (c *OctopusDEGraphQLClient) QueryWithMap(ctx context.Context, result *map[string]interface{}, query string, variables map[string]interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, krakenGraphQLURL_DE, nil)
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.tokenMtx.Lock()
	req.Header.Set("Authorization", *c.token)
	c.tokenMtx.Unlock()

	// Prepare request body
	body := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: variables,
	}

	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
		return fmt.Errorf("error encoding GraphQL request: %w", err)
	}

	req.Body = io.NopCloser(&requestBody)
	req.ContentLength = int64(requestBody.Len())

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse response
	var responseBody struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return fmt.Errorf("error decoding GraphQL response: %w", err)
	}

	// Check for errors
	if len(responseBody.Errors) > 0 {
		errorMessages := make([]string, len(responseBody.Errors))
		for i, err := range responseBody.Errors {
			errorMessages[i] = err.Message
		}
		return fmt.Errorf("GraphQL errors: %v", errorMessages)
	}

	*result = responseBody.Data
	return nil
}

// FetchRates fetches the detailed rate information
func (c *OctopusDEGraphQLClient) FetchRates(accountNumber string) ([]Rate, error) {
	rates, err := c.fetchTimeOfUseRates(accountNumber)
	if err != nil {
		c.log.WARN.Printf("Failed to fetch time-of-use rates: %v. Falling back to simple rates.", err)
		return c.fetchSimpleRates(accountNumber)
	}
	return rates, nil
}

// fetchTimeOfUseRates attempts to fetch detailed time-of-use rates
func (c *OctopusDEGraphQLClient) fetchTimeOfUseRates(accountNumber string) ([]Rate, error) {
	// GraphQL query for rates
	query := `
	query AccountRates($accountNumber: String!) {
		account(accountNumber: $accountNumber) {
			allProperties {
				electricityMalos {
					agreements {
						unitRateInformation {
							__typename
							... on SimpleProductUnitRateInformation {
								latestGrossUnitRateCentsPerKwh
							}
							... on TimeOfUseProductUnitRateInformation {
								rates {
									latestGrossUnitRateCentsPerKwh
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

	c.log.TRACE.Printf("Fetching rates for account: %s", accountNumber)

	// Execute the query
	var result map[string]interface{}
	if err := backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()

		err := c.QueryWithMap(ctx, &result, query, map[string]interface{}{
			"accountNumber": accountNumber,
		})

		if err != nil {
			c.log.ERROR.Printf("GraphQL error: %v", err)
			return err
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)); err != nil {
		return nil, fmt.Errorf("failed to fetch rates: %w", err)
	}

	// Parse result
	var rates []Rate

	// Navigate through response structure
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

	agreement := agreements[0].(map[string]interface{})
	rateInfo, ok := agreement["unitRateInformation"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing rate information")
	}

	// Process rates based on type
	typename, _ := rateInfo["__typename"].(string)
	c.log.TRACE.Printf("Rate type: %s", typename)

	switch typename {
	case "TimeOfUseProductUnitRateInformation":
		ratesArray, ok := rateInfo["rates"].([]interface{})
		if ok && len(ratesArray) > 0 {
			for _, r := range ratesArray {
				rate, ok := r.(map[string]interface{})
				if !ok {
					continue
				}

				// Get price
				rateValue, ok := rate["latestGrossUnitRateCentsPerKwh"].(string)
				if !ok {
					continue
				}

				rateFloat, err := strconv.ParseFloat(rateValue, 64)
				if err != nil {
					c.log.WARN.Printf("Failed to parse rate: %v", err)
					continue
				}

				rateEuros := rateFloat / 100.0
				timeslotName, _ := rate["timeslotName"].(string)

				// Get time slots
				timeSlots, ok := rate["timeslotActivationRules"].([]interface{})
				if !ok || len(timeSlots) == 0 {
					// No slots = all day rate
					rates = append(rates, Rate{
						Price:     rateEuros,
						StartTime: "00:00:00",
						EndTime:   "00:00:00",
						Name:      timeslotName,
					})
					c.log.TRACE.Printf("Added all-day rate: %.4f €/kWh (%s)", rateEuros, timeslotName)
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

					c.log.TRACE.Printf("Added rate: %.4f €/kWh from %s to %s (%s)",
						rateEuros, startTime, endTime, timeslotName)
				}
			}
		}

	case "SimpleProductUnitRateInformation":
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
				c.log.TRACE.Printf("Added standard rate: %.4f €/kWh", rateEuros)
			}
		}
	}

	if len(rates) == 0 {
		return nil, errors.New("no valid rates found")
	}

	return rates, nil
}

// fetchSimpleRates is a fallback for getting basic rate information
func (c *OctopusDEGraphQLClient) fetchSimpleRates(accountNumber string) ([]Rate, error) {
	client := graphql.NewClient(krakenGraphQLURL_DE, http.DefaultClient).
		WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", *c.token)
		})

	var query krakenDEAccountGrossRate
	c.log.TRACE.Printf("Using simple rate query for account: %s", accountNumber)

	// Execute query with retry
	if err := backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()
		return client.Query(ctx, &query, map[string]interface{}{
			"accountNumber": graphql.String(accountNumber),
		})
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)); err != nil {
		c.log.ERROR.Printf("Failed to fetch rates: %v", err)
		return []Rate{{
			Price:     defaultRate,
			StartTime: "00:00:00",
			EndTime:   "00:00:00",
			Name:      "DEFAULT",
		}}, nil
	}

	// Extract rate from response
	grossRate := defaultRate

	// Navigate the response safely
	if len(query.Account.AllProperties) > 0 &&
		len(query.Account.AllProperties[0].ElectricityMalos) > 0 &&
		len(query.Account.AllProperties[0].ElectricityMalos[0].Agreements) > 0 &&
		len(query.Account.AllProperties[0].ElectricityMalos[0].Agreements[0].UnitRateGrossRateInformation) > 0 {

		rateStr := query.Account.AllProperties[0].ElectricityMalos[0].Agreements[0].UnitRateGrossRateInformation[0].GrossRate
		if rateStr != "" {
			rate, err := strconv.ParseFloat(rateStr, 64)
			if err == nil {
				grossRate = rate / 100
				c.log.TRACE.Printf("Found rate: %.4f €/kWh", grossRate)
			}
		}
	}

	return []Rate{{
		Price:     grossRate,
		StartTime: "00:00:00",
		EndTime:   "00:00:00",
		Name:      "STANDARD",
	}}, nil
}

// FetchGrossRates returns a simplified rate (backwards compatibility)
func (c *OctopusDEGraphQLClient) FetchGrossRates(accountNumber string) ([]float64, error) {
	rates, err := c.FetchRates(accountNumber)
	if err != nil {
		return []float64{defaultRate}, nil
	}

	if len(rates) > 0 {
		return []float64{rates[0].Price}, nil
	}

	return []float64{defaultRate}, nil
}
