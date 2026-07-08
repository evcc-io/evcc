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

// ItBaseURI is Octopus Energy Italy's Kraken API root, the same platform under its own domain.
const ItBaseURI = "https://api.oeit-kraken.energy/v1/graphql/"

// Client provides an interface for communicating with an Octopus Energy Kraken platform instance.
type Client struct {
	log *util.Logger
	*graphql.Client
	accountNumber string
}

// NewClient returns a new, authenticated instance for the given Kraken instance
// (other regional Octopus companies run the same platform under their own baseURI).
func NewClient(log *util.Logger, baseURI, email, password, accountNumber string) (*Client, error) {
	ts := oauth2.ReuseTokenSource(nil, &tokenSource{
		log:      log,
		baseURI:  baseURI,
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

	gq := &Client{
		log:           log,
		accountNumber: accountNumber,
		Client:        graphql.NewClient(baseURI, cli),
	}

	return gq, nil
}

// ActiveAgreement queries the Kraken API and returns the active electricity supply agreement.
func (c *Client) ActiveAgreement() (Agreement, error) {
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
