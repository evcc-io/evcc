package graphql

import "time"

// ItAgreement is an electricity supply agreement on Octopus Italy's schema.
// Rates live on the product directly - agreementRates isn't reachable with a customer token.
type ItAgreement struct {
	IsActive  bool
	ValidFrom time.Time
	ValidTo   time.Time
	Product   struct {
		ElectricityProductType `graphql:"... on ElectricityProductType"`
	} `graphql:"product"`
}

// ElectricityProductType is the Italian Kraken schema's product type,
// carrying the customer-facing consumption prices.
type ElectricityProductType struct {
	Code   string
	Prices ElectricityProductPrices
}

// ElectricityProductPrices holds customer-facing prices, already in €/kWh.
// ConsumptionChargeF2/F3 are populated only for time-of-use products.
type ElectricityProductPrices struct {
	ProductType            string
	ConsumptionCharge      string
	ConsumptionChargeF2    string
	ConsumptionChargeF3    string
	ConsumptionChargeUnits string
}

// itGetAgreements uses electricitySupplyPoints (not electricityMalos) and a
// paginated agreements connection (edges/node), unlike Germany's plain list.
type itGetAgreements struct {
	Account struct {
		Properties []struct {
			ElectricitySupplyPoints []struct {
				Pod        string
				Agreements struct {
					Edges []struct {
						Node ItAgreement
					}
				} `graphql:"agreements(first: 10)"`
			}
		}
	} `graphql:"account(accountNumber: $accountNumber)"`
}
