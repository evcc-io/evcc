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

// availability is the state of an instance shared by all its connections
type availability struct {
	seq     uint64 // last sequence number issued to a request
	applied uint64 // sequence number of the request that determined the state
	down    bool
}

var (
	outageMu sync.Mutex
	outages  = make(map[string]*availability)
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

// trackAvailability runs a request to the instance at uri and logs availability changes
// once per instance instead of once per entity. Errors caused by the instance being
// unreachable are marked with api.ErrUnreachable.
func trackAvailability(uri string, req func() error) error {
	seq := nextSequence(uri)
	err := req()
	unreachable := isUnreachable(err)

	// the request did not get a response, e.g. due to missing login
	if _, ok := errors.AsType[*url.Error](err); ok && !unreachable {
		return err
	}

	updateAvailability(uri, seq, unreachable, err)

	if !unreachable {
		return err
	}

	// backoff returns the error wrapped by a permanent error, which would drop the marker
	if pe, ok := errors.AsType[*backoff.PermanentError](err); ok {
		return backoff.Permanent(&unreachableError{pe.Err})
	}

	return &unreachableError{err}
}

// nextSequence numbers requests in the order they are sent
func nextSequence(uri string) uint64 {
	outageMu.Lock()
	defer outageMu.Unlock()

	a, ok := outages[uri]
	if !ok {
		a = new(availability)
		outages[uri] = a
	}

	a.seq++
	return a.seq
}

func updateAvailability(uri string, seq uint64, unreachable bool, err error) {
	outageMu.Lock()
	defer outageMu.Unlock()

	// a slow response must not override the state set by a request sent later
	a := outages[uri]
	if seq < a.applied {
		return
	}
	a.applied = seq

	if a.down == unreachable {
		return
	}
	a.down = unreachable

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
