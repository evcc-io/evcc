package api

import (
	"errors"

	"github.com/cenkalti/backoff/v4"
)

// ErrNotAvailable indicates that a feature is not available
var ErrNotAvailable = errors.New("not available")

// ErrMustRetry indicates that a rate-limited operation should be retried
var ErrMustRetry = errors.New("must retry")

// ErrSponsorRequired indicates that a sponsor token is required
var ErrSponsorRequired = errors.New("sponsorship required, see https://docs.evcc.io/docs/sponsorship")

// ErrMissingCredentials indicates that user/password are missing
var ErrMissingCredentials = backoff.Permanent(errors.New("missing user/password credentials"))

// ErrMissingToken indicates that access/refresh tokens are missing
var ErrMissingToken = backoff.Permanent(errors.New("missing token credentials"))

// ErrOutdated indicates that result is outdated
var ErrOutdated = errors.New("outdated")

// ErrTimeout is the error returned when a timeout happened
var ErrTimeout error = errors.New("timeout")

// LoginRequiredError creates a login error for given auth provider
func LoginRequiredError(providerAuth string) error {
	return backoff.Permanent(&ErrLoginRequired{
		ProviderAuth: providerAuth,
	})
}

// ErrLoginRequired indicates that retrieving tokens credentials waits for login
type ErrLoginRequired struct {
	ProviderAuth string
}

func (err *ErrLoginRequired) Error() string {
	return "login required"
}

// ErrAsleep indicates that vehicle is asleep. Caller may chose to wake up the vehicle and retry.
var ErrAsleep error = errAsleep{}

type errAsleep struct{}

func (errAsleep) Error() string { return "asleep" }
func (errAsleep) Unwrap() error { return ErrTimeout }
