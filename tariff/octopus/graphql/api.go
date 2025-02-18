package graphql

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
)

// BaseURI is Octopus Energy's core API root.
const BaseURI = "https://api.octopus.energy"

// URI is the GraphQL query endpoint for Octopus Energy.
const URI = BaseURI + "/v1/graphql/"

// GermanBaseURI is the German API root.
const GermanBaseURI = "https://api.oeg-kraken.energy"

// GermanURI is the GraphQL query endpoint for the German API.
const GermanURI = GermanBaseURI + "/v1/graphql/"

// OctopusGraphQLClient provides an interface for communicating with Octopus Energy's Kraken platform.
type OctopusGraphQLClient struct {
	*graphql.Client

	// apikey is the Octopus Energy API key (provided by user)
	apikey string

	// token is the GraphQL token used for communication with kraken (we get this ourselves with the apikey)
	token *string
	// tokenExpiration tracks the expiry of the acquired token. A new Token should be obtained if this time is passed.
	tokenExpiration time.Time
	// tokenMtx should be held when requesting a new token.
	tokenMtx sync.Mutex

	// accountNumber is the Octopus Energy account number associated with the given API key (queried ourselves via GraphQL)
	accountNumber string
}

// NewClient returns a new, unauthenticated instance of OctopusGraphQLClient.
func NewClient(log *util.Logger, apikey string) (*OctopusGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusGraphQLClient{
		Client: graphql.NewClient(URI, cli),
		apikey: apikey,
	}

	if err := gq.refreshToken(); err != nil {
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

// NewClientWithEmailPassword returns a new instance of OctopusGraphQLClient authenticated with email and password.
func NewClientWithEmailPassword(log *util.Logger, email, password string) (*OctopusGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusGraphQLClient{
		Client: graphql.NewClient(GermanURI, cli),
	}

	token, err := getKrakenToken(email, password)
	if err != nil {
		return nil, err
	}

	gq.token = &token
	gq.tokenExpiration = time.Now().Add(time.Hour)

	// Future requests must have the appropriate Authorization header set.
	gq.Client = gq.Client.WithRequestModifier(func(r *http.Request) {
		gq.tokenMtx.Lock()
		defer gq.tokenMtx.Unlock()
		r.Header.Add("Authorization", *gq.token)
	})

	return gq, nil
}

// getKrakenToken fetches the token using email and password.
func getKrakenToken(email, password string) (string, error) {
	cli := request.NewHelper(nil)
	var res struct {
		Token string `json:"token"`
	}

	err := cli.Post(GermanURI, struct {
		Query     string `json:"query"`
		Variables struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"variables"`
	}{
		Query: `mutation krakenTokenAuthentication($email: String!, $password: String!) {
                    obtainKrakenToken(input: {email: $email, password: $password}) {
                        token
                    }
                }`,
		Variables: struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}{
			Email:    email,
			Password: password,
		},
	}, &res)

	if err != nil {
		return "", err
	}
	return res.Token, nil
}

// refreshToken updates the GraphQL token from the set apikey.
// Basic caching is provided - it will not update the token if it hasn't expired yet.
func (c *OctopusGraphQLClient) refreshToken() error {
	// take a lock against the token mutex for the refresh
	c.tokenMtx.Lock()
	defer c.tokenMtx.Unlock()

	if time.Until(c.tokenExpiration) > 5*time.Minute {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var q krakenTokenAuthentication
	if err := c.Client.Mutate(ctx, &q, map[string]interface{}{"apiKey": c.apikey}); err != nil {
		return err
	}

	c.token = &q.ObtainKrakenToken.Token
	c.tokenExpiration = time.Now().Add(time.Hour)
	return nil
}

// AccountNumber queries the Account Number assigned to the associated API key.
// Caching is provided.
func (c *OctopusGraphQLClient) AccountNumber() (string, error) {
	// Check cache
	if c.accountNumber != "" {
		return c.accountNumber, nil
	}

	// Update refresh token (if necessary)
	if err := c.refreshToken(); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var q krakenAccountLookup
	if err := c.Client.Query(ctx, &q, nil); err != nil {
		return "", err
	}

	if len(q.Viewer.Accounts) == 0 {
		return "", errors.New("no account associated with given octopus api key")
	}
	if len(q.Viewer.Accounts) > 1 {
		return "", errors.New("more than one octopus account on this api key not supported")
	}
	c.accountNumber = q.Viewer.Accounts[0].Number
	return c.accountNumber, nil
}

// TariffCode queries the Tariff Code of the first Electricity Agreement active on the account.
func (c *OctopusGraphQLClient) TariffCode() (string, error) {
	// Update refresh token (if necessary)
	if err := c.refreshToken(); err != nil {
		return "", err
	}

	// Get Account Number
	acc, err := c.AccountNumber()
	if err != nil {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var q krakenAccountElectricityAgreements
	if err := c.Client.Query(ctx, &q, map[string]interface{}{"accountNumber": acc}); err != nil {
		return "", err
	}

	if len(q.Account.ElectricityAgreements) == 0 {
		return "", errors.New("no electricity agreements found")
	}

	return q.Account.ElectricityAgreements[0].Tariff.TariffCode(), nil
}

// GermanTariffCode queries the Tariff Code of the first Electricity Agreement active on the account for the German API.
func (c *OctopusGraphQLClient) GermanTariffCode() (string, error) {
	// Update refresh token (if necessary)
	if err := c.refreshToken(); err != nil {
		return "", err
	}

	// Get Account Number
	acc, err := c.AccountNumber()
	if err != nil {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var q struct {
		Account struct {
			AllProperties []struct {
				ElectricityMalos []struct {
					Agreements []struct {
						IsActive bool `json:"isActive"`
					} `json:"agreements"`
				} `json:"electricityMalos"`
			} `json:"allProperties"`
		} `json:"account"`
	}

	if err := c.Client.Query(ctx, &q, map[string]interface{}{"accountNumber": acc}); err != nil {
		return "", err
	}

	for _, property := range q.Account.AllProperties {
		for _, malo := range property.ElectricityMalos {
			for _, agreement := range malo.Agreements {
				if agreement.IsActive {
					return "active_tariff_code", nil // Replace with actual tariff code extraction logic
				}
			}
		}
	}

	return "", errors.New("no active electricity agreements found")
}
