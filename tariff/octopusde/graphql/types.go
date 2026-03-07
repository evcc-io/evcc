package graphql

import "time"

// krakenTokenAuthentication is a representation of a GraphQL query for obtaining a Kraken API token.
type krakenTokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {email: $email, password: $password})"`
}

// getDayAheadPrices queries the day-ahead price forecast and the current rate information.
// The unitRateForecast field is only populated for dynamic tariffs.
// The unitRateInformation field covers all tariff types (Simple, TimeOfUse/static).
type getDayAheadPrices struct {
	Account struct {
		Properties []struct {
			ElectricityMalos []struct {
				Agreements []struct {
					IsActive            bool
					ValidFrom           time.Time
					ValidTo             time.Time
					UnitRateInformation agreementUnitRateInformation
					UnitRateForecast    []unitRateForecast
					Product             product
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

// agreementUnitRateInformation is the current rate information for an agreement.
// It supports both SimpleProductUnitRateInformation (fixed rate) and
// TimeOfUseProductUnitRateInformation (time-slot based rates with activation rules).
type agreementUnitRateInformation struct {
	SimpleProductUnitRateInformation    simpleProductUnitRateInformation    `graphql:"... on SimpleProductUnitRateInformation"`
	TimeOfUseProductUnitRateInformation touAgreementUnitRateInformation     `graphql:"... on TimeOfUseProductUnitRateInformation"`
}

// simpleProductUnitRateInformation holds a single fixed rate.
type simpleProductUnitRateInformation struct {
	LatestGrossUnitRateCentsPerKwh string
	NetUnitRateCentsPerKwh         string
}

// touAgreementUnitRateInformation holds multiple time-slot rates with their activation rules.
type touAgreementUnitRateInformation struct {
	Rates []touRate
}

// touRate is a rate with time-slot activation rules (used in non-dynamic ToU agreements).
type touRate struct {
	NetUnitRateCentsPerKwh         string `graphql:"netUnitRateCentsPerKwh"`
	LatestGrossUnitRateCentsPerKwh string `graphql:"latestGrossUnitRateCentsPerKwh"`
	TimeslotName                   string
	TimeslotActivationRules        []timeslotActivationRule
}

// timeslotActivationRule defines the time window during which a rate slot is active.
type timeslotActivationRule struct {
	ActiveFromTime string
	ActiveToTime   string
}

type unitRateForecast struct {
	ValidFrom           time.Time
	ValidTo             time.Time
	UnitRateInformation forecastUnitRateInformation
}

// forecastUnitRateInformation is the rate information embedded in forecast entries.
// Dynamic tariffs use TimeOfUseProductUnitRateInformation; simple forecasts use
// SimpleProductUnitRateInformation.
type forecastUnitRateInformation struct {
	SimpleProductUnitRateInformation    simpleProductUnitRateInformation    `graphql:"... on SimpleProductUnitRateInformation"`
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
