package api

import (
	"errors"
	"net/url"

	"github.com/cenkalti/backoff/v4"
)

// permanentError is a sentinel that backoff treats as permanent without being
// wrapped. backoff.Retry returns PermanentError.Err, so a backoff.Permanent
// sentinel loses its identity on the way out and errors.Is stops matching.
type permanentError struct {
	msg string
}

func (e *permanentError) Error() string {
	return e.msg
}

// As reports the sentinel as *backoff.PermanentError wrapping itself, so
// backoff stops retrying and returns the sentinel unchanged.
func (e *permanentError) As(target any) bool {
	p, ok := target.(**backoff.PermanentError)
	if ok {
		*p = &backoff.PermanentError{Err: e}
	}
	return ok
}

func permanent(msg string) error {
	return &permanentError{msg}
}

// ErrNotAvailable indicates that a feature is not available
var ErrNotAvailable = permanent("not available")

// ErrUnsupportedPlatform indicates unsupported hardware platform
var ErrUnsupportedPlatform = permanent("unsupported platform")

// ErrMustRetry indicates that a rate-limited operation should be retried
var ErrMustRetry = errors.New("must retry")

// ErrSponsorRequired indicates that a sponsor token is required
var ErrSponsorRequired = errors.New("sponsorship required, see https://docs.evcc.io/docs/sponsorship")

// ErrMissingCredentials indicates that user/password are missing
var ErrMissingCredentials = permanent("missing user/password credentials")

// ErrMissingToken indicates that access/refresh tokens are missing
var ErrMissingToken = permanent("missing token credentials")

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
