package octopusde

// GraphQL query types
type krakenDETokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {email: $email, password: $password})"`
}

type krakenDEAccountLookup struct {
	Viewer struct {
		Accounts []struct {
			Number string
		}
	}
}

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

// Rate info structures
type TimeSlotActivationRule struct {
	ActiveFromTime string `json:"activeFromTime"`
	ActiveToTime   string `json:"activeToTime"`
}

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

type TimeOfUseProductUnitRateInformation struct {
	TypeName string            `json:"__typename"`
	Rates    []RateInformation `json:"rates"`
}
