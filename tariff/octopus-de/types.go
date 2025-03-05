package octopusde

// Implementation of intelligent dispatch times is WIP //
// krakenDETokenAuthentication is a representation of a GraphQL query for obtaining a Kraken DE API token.
type krakenDETokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {email: $email, password: $password})"`
}

// krakenDEAccountLookup is a representation of a GraphQL query for obtaining the Account Number associated with the
// credentials used to authorize the request.
type krakenDEAccountLookup struct {
	Viewer struct {
		Accounts []struct {
			Number string
		}
	}
}

// krakenDEAccountGrossRate is a representation of a GraphQL query for obtaining the gross rates for the given account number.
type krakenDEAccountGrossRate struct {
	Account struct {
		AllProperties []struct {
			ElectricityMalos []struct {
				Agreements []struct {
					UnitRateGrossRateInformation []struct {
						GrossRate string
					}
				}
			}
		}
	} `graphql:"account(accountNumber: $accountNumber)"`
}

// TimeSlotActivationRule represents when a specific rate is active
type TimeSlotActivationRule struct {
	ActiveFromTime string `json:"activeFromTime"`
	ActiveToTime   string `json:"activeToTime"`
}

// RateInformation contains details about a specific rate
type RateInformation struct {
	GrossRateInformation []struct {
		Date            string `json:"date"`
		GrossRate       string `json:"grossRate"`
		RateValidToDate string `json:"rateValidToDate"`
		VatRate         string `json:"vatRate"`
	} `json:"grossRateInformation"`
	LatestGrossUnitRateCentsPerKwh string                   `json:"latestGrossUnitRateCentsPerKwh"`
	NetUnitRateCentsPerKwh         string                   `json:"netUnitRateCentsPerKwh"`
	TimeslotActivationRules        []TimeSlotActivationRule `json:"timeslotActivationRules"`
	TimeslotName                   string                   `json:"timeslotName"`
}

// SimpleProductUnitRateInformation represents a simple product with a single rate
type SimpleProductUnitRateInformation struct {
	TypeName             string `json:"__typename"`
	GrossRateInformation []struct {
		Date            string `json:"date"`
		GrossRate       string `json:"grossRate"`
		RateValidToDate string `json:"rateValidToDate"`
		VatRate         string `json:"vatRate"`
	} `json:"grossRateInformation"`
	LatestGrossUnitRateCentsPerKwh string `json:"latestGrossUnitRateCentsPerKwh"`
	NetUnitRateCentsPerKwh         string `json:"netUnitRateCentsPerKwh"`
}

// TimeOfUseProductUnitRateInformation represents a time-of-use product with multiple rates
type TimeOfUseProductUnitRateInformation struct {
	TypeName string            `json:"__typename"`
	Rates    []RateInformation `json:"rates"`
}

// krakenDEAccountRates is a representation of a GraphQL query for obtaining detailed rate information
type krakenDEAccountRates struct {
	Account struct {
		AllProperties []struct {
			ElectricityMalos []struct {
				Agreements []struct {
					UnitRateGrossRateInformation []struct {
						GrossRate string `json:"grossRate"`
					} `json:"unitRateGrossRateInformation"`
					UnitRateInformation struct {
						TypeName                     string                               `json:"__typename"`
						SimpleProductUnitRateInfo    *SimpleProductUnitRateInformation    `json:"... on SimpleProductUnitRateInformation"`
						TimeOfUseProductUnitRateInfo *TimeOfUseProductUnitRateInformation `json:"... on TimeOfUseProductUnitRateInformation"`
					} `json:"unitRateInformation"`
				} `json:"agreements"`
			} `json:"electricityMalos"`
		} `json:"allProperties"`
	} `graphql:"account(accountNumber: $accountNumber)"`
}
