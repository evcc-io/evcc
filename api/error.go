package api

import "errors"

// ErrNotAvailable indicates that a feature is not available
var ErrNotAvailable = errors.New("not available")

// ErrMustRetry indicates that a rate-limited operation should be retried
var ErrMustRetry = errors.New("must retry")

// ErrSponsorRequired indicates that a sponsor token is required
var ErrSponsorRequired = errors.New("sponsorship required, see https://docs.evcc.io/docs/sponsorship")

// ErrMissingCredentials indicates that user/password are missing
var ErrMissingCredentials = errors.New("missing user/password credentials")

// ErrLoginRequired indicates that retrieving tokens credentials waits for login
var ErrLoginRequired = errors.New("login required")

// ErrMissingToken indicates that access/refresh tokens are missing
var ErrMissingToken = errors.New("missing token credentials")

// ErrOutdated indicates that result is outdated
var ErrOutdated = errors.New("outdated")

// ErrTimeout is the error returned when a timeout happened.
// Modeled after context.DeadlineError
var ErrTimeout error = errTimeoutError{}

type errTimeoutError struct{}

func (errTimeoutError) Error() string   { return "timeout" }
func (errTimeoutError) Timeout() bool   { return true }
func (errTimeoutError) Temporary() bool { return true }

// ErrAsleep indicates that vehicle is asleep. Caller may chose to wake up the vehicle and retry.
var ErrAsleep error = errAsleep{}

type errAsleep struct{}

func (errAsleep) Error() string { return "asleep" }
func (errAsleep) Unwrap() error { return ErrTimeout }
