package graphql

// BaseURI is Octopus Energy's core API root.
const BaseURI = "https://api.octopus.energy"

// URI is the GraphQL query endpoint for Octopus Energy.
const URI = BaseURI + "/v1/graphql/"

// FIXME these don't need to be public

// KrakenTokenAuthentication is a representation of a GraphQL query for obtaining a Kraken API token.
type KrakenTokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {APIKey: $apiKey})"`
}

// KrakenAccountLookup is a representation of a GraphQL query for obtaining the Account Number associated with the
// credentials used to authorize the request.
type KrakenAccountLookup struct {
	Viewer struct {
		Accounts []struct {
			Number string
		}
	}
}

type tariffType struct {
	Id                   string
	DisplayName          string
	FullName             string
	ProductCode          string
	StandingCharge       float32
	PreVatStandingCharge float32
}

type tariffTypeWithTariffCode struct {
	tariffType
	TariffCode string
}

type StandardTariff struct {
	tariffTypeWithTariffCode
}
type DayNightTariff struct {
	tariffTypeWithTariffCode
}
type ThreeRateTariff struct {
	tariffTypeWithTariffCode
}
type HalfHourlyTariff struct {
	tariffTypeWithTariffCode
}
type PrepayTariff struct {
	tariffTypeWithTariffCode
}

type KrakenAccountElectricityAgreements struct {
	Account struct {
		ElectricityAgreements []struct {
			Id     int
			Tariff struct {
				// yukky but the best way I can think of to handle this
				// access via any relevant tariff data entry (i.e. StandardTariff)
				// TODO would appreciate peer review
				StandardTariff   `graphql:"... on StandardTariff"`
				DayNightTariff   `graphql:"... on DayNightTariff"`
				ThreeRateTariff  `graphql:"... on ThreeRateTariff"`
				HalfHourlyTariff `graphql:"... on HalfHourlyTariff"`
				PrepayTariff     `graphql:"... on PrepayTariff"`
			}
			MeterPoint struct {
				// Mpan is the serial number of the meter that this ElectricityAgreement is bound to.
				Mpan string
			}
		} `graphql:"electricityAgreements(active: true)"`
	} `graphql:"account(accountNumber: $accountNumber)"`
}
