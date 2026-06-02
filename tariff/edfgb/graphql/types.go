package graphql

import "time"

// krakenTokenAuthentication is used to obtain a Kraken API token via email/password.
type krakenTokenAuthentication struct {
	ObtainKrakenToken struct {
		Token string
	} `graphql:"obtainKrakenToken(input: {email: $email, password: $password})"`
}

// getMeterPoint fetches the MPAN (meter point reference) for the account.
type getMeterPoint struct {
	Account struct {
		Properties []struct {
			ElectricityMeterPoints []struct {
				Mpan string
			}
		}
	} `graphql:"account(accountNumber: $accountNumber)"`
}

// getApplicableRates fetches time-bounded rate data for a given meter point.
type getApplicableRates struct {
	ApplicableRates struct {
		Edges []struct {
			Node ApplicableRate
		}
	} `graphql:"applicableRates(accountNumber: $accountNumber, mpxn: $mpxn, startAt: $startAt, endAt: $endAt)"`
}

// ApplicableRate is a single rate period returned by the applicableRates query.
type ApplicableRate struct {
	ValidFrom                time.Time
	ValidTo                  time.Time
	GrossUnitRateCentsPerKwh string
	NetUnitRateCentsPerKwh   string
}
