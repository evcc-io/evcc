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

// OctopusGraphQLClient provides an interface for communicating with Octopus Energy's Kraken platform.
type OctopusGraphQLClient struct {
	*graphql.Client

	// Local logging utility.
	log *util.Logger

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

	// accountNumberDesire is an optional Octopus Energy account number to search for, if there are multiple accounts on the key.
	accountNumberDesire string
}

// NewClient returns a new, unauthenticated instance of OctopusGraphQLClient.
func NewClient(log *util.Logger, apikey string, accountNumber string) (*OctopusGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusGraphQLClient{
		Client:              graphql.NewClient(URI, cli),
		log:                 log,
		apikey:              apikey,
		accountNumberDesire: accountNumber,
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
	c.log.TRACE.Println("GraphQL: refreshed token, now expires", c.tokenExpiration)
	return nil
}

// AccountNumber queries the Account Number assigned to the associated API key.
// Caching is provided.
// If more than one Account is bound to the API Key, this will search for AccountNumberDesire in the list of available accounts,
// and return an error if it cannot be found.
func (c *OctopusGraphQLClient) AccountNumber() (accountNumber string, err error) {
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

	c.accountNumber, err = filterAccount(q.Viewer.Accounts, c.accountNumberDesire)
	if err != nil {
		if errors.Is(err, ErrMultipleAccounts) {
			c.log.ERROR.Println("There is more than one account associated with this Octopus API key.")
			c.log.ERROR.Println("Please add one of the following accounts to your tariff configuration under the accountNumber key:")
			for _, account := range q.Viewer.Accounts {
				c.log.ERROR.Println(" - ", account.Number)
			}
		}
		return "", err
	}

	c.log.TRACE.Println("GraphQL: using account number:", c.accountNumber)
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
		return "", err
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

	// check type
	//switch t := q.Account.ElectricityAgreements[0].Tariff.(type) {
	//
	//}
	tariffCode := q.Account.ElectricityAgreements[0].Tariff.TariffCode()
	c.log.TRACE.Println("GraphQL: tariff code found:", tariffCode)

	return tariffCode, nil
}

// filterAccount searches the given accounts for one exactly matching the desire.
// If a desire is set, but cannot be found, it will return an error.
// If a desire is not set, but there is more than one account, it will return an error.
// If a desire is not set, but there is only one account, it will return the Number of that account.
func filterAccount(accounts []krakenAccount, desire string) (result string, err error) {
	// Test for no available accounts.
	if len(accounts) == 0 {
		return "", ErrNoAccounts
	}

	// If a desired account number is set, let's try and bind to that first.
	if desire != "" {
		for _, account := range accounts {
			if account.Number == desire {
				return account.Number, nil
			}
		}
		// A Desire was set, but we couldn't find it.
		return "", ErrAccountNotFound
	}

	if len(accounts) == 1 {
		// Only one possible result, filtration not enabled.
		return accounts[0].Number, nil
	}

	// There is more than one account, and no filter is set. We need the user to intervene at this point, as we can't presume.
	return "", ErrMultipleAccounts
}
