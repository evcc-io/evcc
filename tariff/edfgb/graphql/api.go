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

// BaseURI is EDF UK's Kraken API root.
const BaseURI = "https://api.edfgb-kraken.energy/v1/graphql/"

// EdfGbGraphQLClient communicates with EDF UK's Kraken platform.
type EdfGbGraphQLClient struct {
	log           *util.Logger
	*graphql.Client
	accountNumber string
}

// NewClient returns a new, authenticated EdfGbGraphQLClient.
func NewClient(log *util.Logger, email, password, accountNumber string) (*EdfGbGraphQLClient, error) {
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

	gq := &EdfGbGraphQLClient{
		log:           log,
		accountNumber: accountNumber,
		Client:        graphql.NewClient(BaseURI, cli),
	}

	return gq, nil
}

// MPAN fetches the electricity meter point reference number for this account.
func (c *EdfGbGraphQLClient) MPAN() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var q getMeterPoint
	if err := c.Client.Query(ctx, &q, map[string]any{
		"accountNumber": c.accountNumber,
	}); err != nil {
		return "", err
	}

	for _, prop := range q.Account.Properties {
		for _, mp := range prop.ElectricityMeterPoints {
			if mp.Mpan != "" {
				return mp.Mpan, nil
			}
		}
	}

	return "", errors.New("no electricity meter point found")
}

// Rates fetches applicable rates for the given MPAN and time window.
func (c *EdfGbGraphQLClient) Rates(mpan string, startAt, endAt time.Time) ([]ApplicableRate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var q getApplicableRates
	if err := c.Client.Query(ctx, &q, map[string]any{
		"accountNumber": c.accountNumber,
		"mpxn":          mpan,
		"startAt":       startAt,
		"endAt":         endAt,
	}); err != nil {
		return nil, err
	}

	rates := make([]ApplicableRate, 0, len(q.ApplicableRates.Edges))
	for _, edge := range q.ApplicableRates.Edges {
		rates = append(rates, edge.Node)
	}
	return rates, nil
}
