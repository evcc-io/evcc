package graphql

import (
	"context"
	"errors"
	"time"
)

// ItActiveAgreement queries the Italian Kraken API and returns the active
// electricity supply agreement, including its product's resolved prices.
func (c *Client) ItActiveAgreement() (ItAgreement, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var q itGetAgreements
	if err := c.Client.Query(ctx, &q, map[string]any{
		"accountNumber": c.accountNumber,
	}); err != nil {
		return ItAgreement{}, err
	}

	for _, property := range q.Account.Properties {
		for _, sp := range property.ElectricitySupplyPoints {
			for _, edge := range sp.Agreements.Edges {
				if edge.Node.IsActive {
					return edge.Node, nil
				}
			}
		}
	}

	return ItAgreement{}, errors.New("no active agreement found")
}
