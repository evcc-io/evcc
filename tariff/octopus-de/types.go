package octopusde

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/hasura/go-graphql-client"
)

// krakenDETokenAuthentication is a representation of a GraphQL query for obtaining a Kraken API token.
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
		AllProperties struct {
			ElectricityMalos struct {
				Agreements []struct {
					UnitRateGrossRateInformation struct {
						GrossRate float64
					}
				}
			}
		}
	} `graphql:"account(accountNumber: $accountNumber)"`
}

// OctopusDEGraphQLClient provides an interface for communicating with Octopus Energy's Kraken platform.
type OctopusDEGraphQLClient struct {
	*graphql.Client

	// email and password are the Octopus Energy account credentials (provided by user)
	email    string
	password string

	// token is the GraphQL token used for communication with kraken (we get this ourselves with the email and password)
	token *string
	// tokenExpiration tracks the expiry of the acquired token. A new Token should be obtained if this time is passed.
	tokenExpiration time.Time
	// tokenMtx should be held when requesting a new token.
	tokenMtx sync.Mutex

	// accountNumber is the Octopus Energy account number associated with the given credentials (queried ourselves via GraphQL)
	accountNumber string

	// timeout is the duration for which the client will wait for a response from the server.
	timeout time.Duration

	// log is the logger used for logging messages.
	log *util.Logger
}
