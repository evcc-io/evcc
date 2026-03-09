package graphql

import "time"

// krakenTokenAuthentication is a representation of a GraphQL query for obtaining a Kraken API token.
type krakenTokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {email: $email, password: $password})"`
}

// Agreement represents a single electricity supply agreement.
// The unitRateForecast field is only populated for dynamic tariffs.
// The unitRateInformation field covers all tariff types (Simple, TimeOfUse).
type Agreement struct {
	IsActive            bool
	ValidFrom           time.Time
	ValidTo             time.Time
	UnitRateInformation AgreementUnitRateInformation
	UnitRateForecast    []UnitRateForecast
	Product             product
}

type getAgreements struct {
	Account struct {
		Properties []struct {
			ElectricityMalos []struct {
				Agreements []Agreement
			}
		}
	} `graphql:"account(accountNumber: $accountNumber)"`
}

type product struct {
	Code        string
	IsTimeOfUse bool
	Term        int
}

// AgreementUnitRateInformation is the current rate information for an agreement.
// It supports both SimpleProductUnitRateInformation (fixed rate) and
// TimeOfUseProductUnitRateInformation (time-slot based rates with activation rules).
type AgreementUnitRateInformation struct {
	SimpleProductUnitRateInformation    SimpleProductUnitRateInformation `graphql:"... on SimpleProductUnitRateInformation"`
	TimeOfUseProductUnitRateInformation TouAgreementUnitRateInformation  `graphql:"... on TimeOfUseProductUnitRateInformation"`
}

// SimpleProductUnitRateInformation holds a single fixed rate.
type SimpleProductUnitRateInformation struct {
	LatestGrossUnitRateCentsPerKwh string
	NetUnitRateCentsPerKwh         string
}

// TouAgreementUnitRateInformation holds multiple time-slot rates with their activation rules.
type TouAgreementUnitRateInformation struct {
	Rates []TouRate
}

// TouRate is a rate with time-slot activation rules (used in non-dynamic ToU agreements).
type TouRate struct {
	NetUnitRateCentsPerKwh         string `graphql:"netUnitRateCentsPerKwh"`
	LatestGrossUnitRateCentsPerKwh string `graphql:"latestGrossUnitRateCentsPerKwh"`
	TimeslotName                   string
	TimeslotActivationRules        []TimeslotActivationRule
}

// TimeslotActivationRule defines the time window during which a rate slot is active.
type TimeslotActivationRule struct {
	ActiveFromTime string
	ActiveToTime   string
}

// UnitRateForecast holds a single forecast entry with its validity window.
type UnitRateForecast struct {
	ValidFrom           time.Time
	ValidTo             time.Time
	UnitRateInformation ForecastUnitRateInformation
}

// ForecastUnitRateInformation is the rate information embedded in forecast entries.
// Dynamic tariffs use TimeOfUseProductUnitRateInformation; simple forecasts use
// SimpleProductUnitRateInformation.
type ForecastUnitRateInformation struct {
	SimpleProductUnitRateInformation    SimpleProductUnitRateInformation    `graphql:"... on SimpleProductUnitRateInformation"`
	TimeOfUseProductUnitRateInformation TimeOfUseProductUnitRateInformation `graphql:"... on TimeOfUseProductUnitRateInformation"`
}

// TimeOfUseProductUnitRateInformation holds a list of per-slot rates for dynamic/ToU forecasts.
type TimeOfUseProductUnitRateInformation struct {
	Rates []Rate
}

// Rate holds the net and gross unit rate strings for a single dynamic forecast slot.
type Rate struct {
	NetUnitRateCentsPerKwh         string `graphql:"netUnitRateCentsPerKwh"`
	LatestGrossUnitRateCentsPerKwh string `graphql:"latestGrossUnitRateCentsPerKwh"`
}
