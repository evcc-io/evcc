package graphql

import (
	"context"
	"errors"
	"net/http"
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

// ActiveAgreement queries the Kraken API and returns the active electricity supply agreement.
func (c *OctopusDeGraphQLClient) ActiveAgreement() (Agreement, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var q getAgreements
	if err := c.Client.Query(ctx, &q, map[string]any{
		"accountNumber": c.accountNumber,
	}); err != nil {
		return Agreement{}, err
	}

	if len(q.Account.Properties) == 0 {
		return Agreement{}, errors.New("no properties found")
	}

	agr, err := findActiveAgreement(&q)
	if err != nil {
		return Agreement{}, err
	}

	return *agr, nil
}

// findActiveAgreement returns the first agreement marked IsActive across all properties.
func findActiveAgreement(q *getAgreements) (*Agreement, error) {
	for _, property := range q.Account.Properties {
		for _, malo := range property.ElectricityMalos {
			for _, agr := range malo.Agreements {
				if agr.IsActive {
					return &agr, nil
				}
			}
		}
	}
	return nil, errors.New("no active agreement found")
}
