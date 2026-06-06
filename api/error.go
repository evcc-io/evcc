package api

import (
	"errors"
	"net/url"

	"github.com/cenkalti/backoff/v4"
)

// ErrNotAvailable indicates that a feature is not available
var ErrNotAvailable = backoff.Permanent(errors.New("not available"))

// ErrUnsupportedPlatform indicates unsupported hardware platform
var ErrUnsupportedPlatform error = backoff.Permanent(errors.New("unsupported platform"))

// ErrMustRetry indicates that a rate-limited operation should be retried
var ErrMustRetry = errors.New("must retry")

// ErrSponsorRequired indicates that a sponsor token is required
var ErrSponsorRequired = errors.New("sponsorship required, see https://docs.evcc.io/docs/sponsorship")

// ErrMissingCredentials indicates that required static credentials are absent in
// configuration and the operation cannot proceed without user intervention.
// This error is marked permanent to stop retry loops.
var ErrMissingCredentials = backoff.Permanent(errors.New("missing user/password credentials"))

// ErrCredentialsRequired indicates that credentials must be provided now
// (for example, first login when no persisted authentication is available).
// Unlike ErrMissingCredentials, this is not marked permanent so callers can
// surface an actionable credential prompt.
var ErrCredentialsRequired = errors.New("credentials required")

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

// ErrUrl indicates that the error contains an extractable URL
type ErrUrl struct {
	err string
	url *url.URL
}

// UrlError creates an error message containing an url
func UrlError(err string, url *url.URL) *ErrUrl {
	return &ErrUrl{err, url}
}

func (err *ErrUrl) Error() string {
	return err.err
}

func (err *ErrUrl) URL() *url.URL {
	return err.url
}

// ErrAsleep indicates that vehicle is asleep. Caller may chose to wake up the vehicle and retry.
var ErrAsleep error = errAsleep{}

type errAsleep struct{}

func (errAsleep) Error() string { return "asleep" }
func (errAsleep) Unwrap() error { return ErrTimeout }
