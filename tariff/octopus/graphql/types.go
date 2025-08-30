package graphql

// krakenTokenAuthentication is a representation of a GraphQL query for obtaining a Kraken API token.
type krakenTokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {APIKey: $apiKey})"`
}

// krakenAccountLookup is a representation of a GraphQL query for obtaining the Account Number associated with the
// credentials used to authorize the request.
type krakenAccountLookup struct {
	Viewer struct {
		Accounts []krakenAccount
	}
}

// krakenAccount represents an Octopus Energy account.
type krakenAccount struct {
	Number string
}

type tariffData struct {
	// yukky but the best way I can think of to handle this
	// access via any relevant tariff data entry (i.e. standardTariff)
	standardTariff   `graphql:"... on StandardTariff"`
	dayNightTariff   `graphql:"... on DayNightTariff"`
	threeRateTariff  `graphql:"... on ThreeRateTariff"`
	halfHourlyTariff `graphql:"... on HalfHourlyTariff"`
	prepayTariff     `graphql:"... on PrepayTariff"`
}

// TariffCode is a shortcut function to obtaining the Tariff Code of the given tariff, regardless of tariff type.
// Developer Note: GraphQL query returns the same element keys regardless of type,
// so it should always be decoded as standardTariff at least.
// We are unlikely to use the other Tariff types for data access (?).
func (d *tariffData) TariffCode() string {
	return d.standardTariff.TariffCode
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

type standardTariff struct {
	tariffTypeWithTariffCode
}
type dayNightTariff struct {
	tariffTypeWithTariffCode
}
type threeRateTariff struct {
	tariffTypeWithTariffCode
}
type halfHourlyTariff struct {
	tariffTypeWithTariffCode
}
type prepayTariff struct {
	tariffTypeWithTariffCode
}

type krakenAccountElectricityAgreements struct {
	Account struct {
		ElectricityAgreements []struct {
			Id         int
			Tariff     tariffData
			MeterPoint struct {
				// Mpan is the serial number of the meter that this ElectricityAgreement is bound to.
				Mpan string
			}
		} `graphql:"electricityAgreements(active: true)"`
	} `graphql:"account(accountNumber: $accountNumber)"`
}
