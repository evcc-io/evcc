package graphql

import "time"

// krakenTokenAuthentication is a representation of a GraphQL query for obtaining a Kraken API token.
type krakenTokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {email: $email, password: $password})"`
}

// getDayAheadPrices queries the day-ahead price forecast
type getDayAheadPrices struct {
	Account struct {
		Properties []struct {
			ElectricityMalos []struct {
				Agreements []struct {
					IsActive         bool
					UnitRateForecast []unitRateForecast
					Product          product
				}
			}
		}
	} `graphql:"account(accountNumber: $accountNumber)"`
}

type product struct {
	Code        string
	IsTimeOfUse bool
	Term        int
}

type unitRateForecast struct {
	ValidFrom           time.Time
	ValidTo             time.Time
	UnitRateInformation unitRateInformation
}

type unitRateInformation struct {
	TimeOfUseProductUnitRateInformation timeOfUseProductUnitRateInformation `graphql:"... on TimeOfUseProductUnitRateInformation"`
}

type timeOfUseProductUnitRateInformation struct {
	Rates []rate
}

type rate struct {
	NetUnitRateCentsPerKwh         string `graphql:"netUnitRateCentsPerKwh"`
	LatestGrossUnitRateCentsPerKwh string `graphql:"latestGrossUnitRateCentsPerKwh"`
}

// RatePeriod represents a rate period with pricing information
type RatePeriod struct {
	ValidFrom                      time.Time
	ValidTo                        time.Time
	NetUnitRateCentsPerKwh         float64
	LatestGrossUnitRateCentsPerKwh float64
}
