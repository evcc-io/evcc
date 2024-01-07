package graphql

import (
	"context"
	"errors"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
	"net/http"
	"sync"
	"time"
)

type OctopusGraphQLClient struct {
	*graphql.Client
	log *util.Logger

	// apikey is the Octopus Energy API key (provided by user)
	apikey string
	// token is the GraphQL token used for communication with kraken (we get this ourselves with the apikey)
	token *string
	// accountNumber is the Octopus Energy account number associated with the given API key (queried ourselves via GraphQL)
	accountNumber   string
	tokenExpiration time.Time
	tokenMtx        sync.Mutex
}

func NewClient(log *util.Logger, apikey string) (*OctopusGraphQLClient, error) {
	cli := request.NewClient(log)

	gq := &OctopusGraphQLClient{
		Client: graphql.NewClient(URI, cli),
		log:    log,
		apikey: apikey,
	}

	err := gq.refreshToken()
	if err != nil {
		return nil, err
	}
	// Future requests must have the appropriate Authorization header set.
	reqMod := graphql.RequestModifier(
		func(r *http.Request) {
			r.Header.Add("Authorization", *gq.token)
		})
	gq.Client = gq.Client.WithRequestModifier(reqMod)

	return gq, err
}

// refreshToken updates the GraphQL token from the set apikey.
// Basic caching is provided - it will not update the token if it hasn't expired yet.
func (c *OctopusGraphQLClient) refreshToken() error {
	now := time.Now()
	if !c.tokenExpiration.IsZero() && c.tokenExpiration.After(now) {
		c.log.TRACE.Print("using cached octopus token")
		return nil
	}

	// TODO is this a good use of background context?
	ctx := context.Background()
	// take a lock against the token mutex for the refresh
	c.tokenMtx.Lock()
	defer c.tokenMtx.Unlock()

	var q KrakenTokenAuthentication
	err := c.Client.Mutate(ctx, &q, map[string]interface{}{"apiKey": c.apikey})
	if err != nil {
		return err
	}
	c.log.INFO.Println("got GQL token from octopus")
	c.token = &q.ObtainKrakenToken.Token
	// Refresh in 55 minutes (the token lasts an hour, but just to be safe...)
	c.tokenExpiration = time.Now().Add(time.Minute * 55)
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

	// TODO is this a good use of background context?
	ctx := context.Background()

	var q KrakenAccountLookup
	err := c.Client.Query(ctx, &q, nil)
	if err != nil {
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

	// TODO is this a good use of background context?
	ctx := context.Background()

	var q KrakenAccountElectricityAgreements
	err = c.Client.Query(ctx, &q, map[string]interface{}{"accountNumber": acc})
	if err != nil {
		return "", err
	}

	if len(q.Account.ElectricityAgreements) == 0 {
		return "", errors.New("no electricity agreements found")
	}

	// check type
	//switch t := q.Account.ElectricityAgreements[0].Tariff.(type) {
	//
	//}
	return q.Account.ElectricityAgreements[0].Tariff.StandardTariff.TariffCode, nil
}
