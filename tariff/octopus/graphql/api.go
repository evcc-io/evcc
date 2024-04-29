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
}

// NewClient returns a new, unauthenticated instance of OctopusGraphQLClient.
func NewClient(log *util.Logger, apikey string) (*OctopusGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusGraphQLClient{
		Client: graphql.NewClient(URI, cli),
		log:    log,
		apikey: apikey,
	}

	if err := gq.refreshToken(); err != nil {
		return nil, err
	}

	// Future requests must have the appropriate Authorization header set.
	reqMod := graphql.RequestModifier(
		func(r *http.Request) {
			r.Header.Add("Authorization", *gq.token)
		})
	gq.Client = gq.Client.WithRequestModifier(reqMod)

	return gq, nil
}

// refreshToken updates the GraphQL token from the set apikey.
// Basic caching is provided - it will not update the token if it hasn't expired yet.
func (c *OctopusGraphQLClient) refreshToken() error {
	// take a lock against the token mutex for the refresh
	c.tokenMtx.Lock()
	defer c.tokenMtx.Unlock()

	if time.Until(c.tokenExpiration) > 5*time.Minute {
		c.log.TRACE.Print("using cached octopus token")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var q krakenTokenAuthentication
	if err := c.Client.Mutate(ctx, &q, map[string]interface{}{"apiKey": c.apikey}); err != nil {
		return err
	}
	c.log.TRACE.Println("got GQL token from octopus")
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
		c.log.WARN.Print("more than one octopus account on this api key - picking the first one. please file an issue!")
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

	// check type
	//switch t := q.Account.ElectricityAgreements[0].Tariff.(type) {
	//
	//}
	return q.Account.ElectricityAgreements[0].Tariff.TariffCode(), nil
}
