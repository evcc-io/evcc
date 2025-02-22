package octopusde

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
)

// rakenGraphQLURL_DE is the GraphQL query endpoint for Octopus Energy Germany.
const krakenGraphQLURL_DE = "https://api.oeg-kraken.energy/v1/graphql/"

// NewClient returns a new, unauthenticated instance of OctopusGraphQLClient.
func NewClient(log *util.Logger, email, password string) (*OctopusDEGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusDEGraphQLClient{
		Client:   graphql.NewClient(krakenGraphQLURL_DE, cli),
		email:    email,
		password: password,
		timeout:  10 * time.Second,
		log:      log,
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

// FetchGrossRates fetches the gross rates for the given account number.
func (c *OctopusDEGraphQLClient) FetchGrossRates(accountNumber string) ([]float64, error) {
	client := graphql.NewClient(krakenGraphQLURL_DE, http.DefaultClient).
		WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", *c.token)
		})

	var query krakenDEAccountGrossRate

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
		return nil
	}, backoff.NewExponentialBackOff()); err != nil {
		c.log.ERROR.Printf("Backoff error: %v", err)
		return nil, err
	}

	// Convert grossRate from cents to euros
	grossRates := make([]float64, len(query.Account.AllProperties.ElectricityMalos.Agreements))
	for i, agreement := range query.Account.AllProperties.ElectricityMalos.Agreements {
		grossRates[i] = agreement.UnitRateGrossRateInformation.GrossRate / 100
	}

	return grossRates, nil
}
