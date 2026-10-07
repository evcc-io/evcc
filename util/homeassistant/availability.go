package homeassistant

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
)

var (
	outageMu sync.Mutex
	outages  = make(map[string]bool)
)

// unreachableError marks a request error caused by the instance being unreachable
type unreachableError struct {
	error
}

func (e *unreachableError) Unwrap() []error {
	return []error{e.error, api.ErrUnreachable}
}

// isUnreachable reports whether err indicates that the instance could not be reached
func isUnreachable(err error) bool {
	if se, ok := errors.AsType[*request.StatusError](err); ok {
		// returned by the supervisor or a reverse proxy while Home Assistant is down
		return se.HasStatus(http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout)
	}

	if _, ok := errors.AsType[*net.OpError](err); ok {
		return true
	}

	ne, ok := errors.AsType[net.Error](err)
	return ok && ne.Timeout()
}

// trackAvailability logs availability changes once per instance instead of once per entity.
// Errors caused by the instance being unreachable are marked with api.ErrUnreachable.
func trackAvailability(uri string, err error) error {
	unreachable := isUnreachable(err)

	// the request did not get a response, e.g. due to missing login
	if _, ok := errors.AsType[*url.Error](err); ok && !unreachable {
		return err
	}

	outageMu.Lock()
	defer outageMu.Unlock()

	if outages[uri] != unreachable {
		outages[uri] = unreachable

		if unreachable {
			cause := err
			if ue, ok := errors.AsType[*url.Error](err); ok {
				cause = ue.Err
			}
			log.ERROR.Printf("Home Assistant unavailable at %s: %v", uri, cause)
		} else {
			log.INFO.Printf("Home Assistant available again at %s", uri)
		}
	}

	if !unreachable {
		return err
	}

	// backoff returns the error wrapped by a permanent error, which would drop the marker
	if pe, ok := errors.AsType[*backoff.PermanentError](err); ok {
		return backoff.Permanent(&unreachableError{pe.Err})
	}

	return &unreachableError{err}
}
