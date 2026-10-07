package homeassistant

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util/request"
)

// retryDelay limits requests to an unavailable instance to one per period
const retryDelay = 5 * time.Second

// availability is the state of an instance shared by all its connections
type availability struct {
	seq     uint64    // last sequence number issued to a request
	applied uint64    // sequence number of the request that determined the state
	err     error     // cause of the outage, nil while available
	retry   time.Time // during an outage, requests fail without network access until then
}

var (
	outageMu sync.Mutex
	outages  = make(map[string]*availability)

	// availability changes in the order they happened, logged by one goroutine at a time
	changes  []func()
	draining bool
)

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

// trackAvailability runs a request to the instance at uri. While the instance is
// unavailable, requests fail without network access except for one per retryDelay.
// Errors caused by the instance being unreachable are permanent, since retrying
// them immediately only adds load and delay.
func trackAvailability(uri string, req func() error) error {
	seq, err := beginRequest(uri)
	if err != nil {
		return backoff.Permanent(err)
	}

	err = req()
	unreachable := isUnreachable(err)

	// the request did not get a response, e.g. due to missing login
	if _, ok := errors.AsType[*url.Error](err); ok && !unreachable {
		return err
	}

	updateAvailability(uri, seq, unreachable, err)

	if !unreachable {
		return err
	}

	// status errors are already permanent
	if _, ok := errors.AsType[*backoff.PermanentError](err); ok {
		return err
	}

	return backoff.Permanent(err)
}

// beginRequest numbers requests in the order they are sent, or returns the
// outage cause if the request must not be sent
func beginRequest(uri string) (uint64, error) {
	outageMu.Lock()
	defer outageMu.Unlock()

	a, ok := outages[uri]
	if !ok {
		a = new(availability)
		outages[uri] = a
	}

	if a.err != nil {
		if time.Now().Before(a.retry) {
			return 0, a.err
		}

		// let this request through and keep failing the others for another retryDelay
		a.retry = time.Now().Add(retryDelay)
	}

	a.seq++
	return a.seq, nil
}

func updateAvailability(uri string, seq uint64, unreachable bool, err error) {
	cause := err
	if ue, ok := errors.AsType[*url.Error](err); ok {
		cause = ue.Err
	}

	setAvailability(uri, seq, unreachable, cause)
	logChanges()
}

// setAvailability updates the instance state and queues a log line if its availability changed
func setAvailability(uri string, seq uint64, unreachable bool, cause error) {
	outageMu.Lock()
	defer outageMu.Unlock()

	// a slow response must not override the state set by a request sent later
	a := outages[uri]
	if seq < a.applied {
		return
	}
	a.applied = seq

	if !unreachable {
		if a.err != nil {
			a.err = nil
			changes = append(changes, func() { log.INFO.Printf("Home Assistant available again at %s", uri) })
		}
		return
	}

	if a.err == nil {
		changes = append(changes, func() { log.ERROR.Printf("Home Assistant unavailable at %s: %v", uri, cause) })
	}

	a.err = fmt.Errorf("%s unavailable: %w", uri, cause)
	a.retry = time.Now().Add(retryDelay)
}

// logChanges logs queued availability changes in order. The lock is released while
// logging since writing a log line may block. While another goroutine is logging,
// callers return immediately and their changes are logged by that goroutine.
func logChanges() {
	outageMu.Lock()
	defer outageMu.Unlock()

	if draining {
		return
	}
	draining = true

	for len(changes) > 0 {
		next := changes[0]
		changes = changes[1:]

		outageMu.Unlock()
		next()
		outageMu.Lock()
	}

	changes = nil
	draining = false
}
