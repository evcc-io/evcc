package easee

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// CommandDispatcher owns the full lifecycle of an Easee command:
// HTTP POST → response parsing → SignalR CommandResponse correlation.
type CommandDispatcher struct {
	helper          *request.Helper
	mu              sync.Mutex
	pendingTicks    map[int64]chan SignalRCommandResponse
	pendingByID     map[ObservationID]chan SignalRCommandResponse
	expectedOrphans map[ObservationID]int
	log             *util.Logger
	timeout         time.Duration
}

// NewCommandDispatcher creates a dispatcher. helper must be the authenticated
// HTTP client used for all Easee API calls.
func NewCommandDispatcher(helper *request.Helper, log *util.Logger, timeout time.Duration) *CommandDispatcher {
	return &CommandDispatcher{
		helper:          helper,
		log:             log,
		timeout:         timeout,
		pendingTicks:    make(map[int64]chan SignalRCommandResponse),
		pendingByID:     make(map[ObservationID]chan SignalRCommandResponse),
		expectedOrphans: make(map[ObservationID]int),
	}
}

// Dispatch routes an incoming CommandResponse to the appropriate waiter.
// Must be called from the Easee.CommandResponse SignalR handler.
// Logs a WARN if no pending registration or expected orphan matches.
func (d *CommandDispatcher) Dispatch(res SignalRCommandResponse) {
	obsID := ObservationID(res.ID)

	d.mu.Lock()
	chTick, tickOk := d.pendingTicks[res.Ticks]
	chID, idOk := d.pendingByID[obsID]
	d.mu.Unlock()

	// Tick lookup takes priority over ID lookup (primary correlation).
	// ID lookup is a fallback for backend clock drift / load balancer
	// scenarios where the delivered Ticks differs from the HTTP 202 body.
	if tickOk {
		// Channels are buffered (capacity 1) — this send never blocks even if
		// the waiter has timed out and unregistered the channel already.
		chTick <- res
		return
	}

	if idOk {
		chID <- res // buffered (capacity 1), see comment above
		return
	}

	if d.CancelOrphan(obsID) {
		return
	}

	d.log.WARN.Printf("rogue CommandResponse: charger %s ObservationID=%s Ticks=%d "+
		"(accepted=%v, resultCode=%d) which was not triggered by evcc — "+
		"another system may be controlling this charger",
		res.SerialNumber, obsID, res.Ticks, res.WasAccepted, res.ResultCode)
}

// ExpectOrphan pre-registers one expected CommandResponse per id for a
// sync (HTTP 200) endpoint that still produces a CommandResponse on the wire.
// Must be called before Send to avoid a race with the arriving CommandResponse.
func (d *CommandDispatcher) ExpectOrphan(ids ...ObservationID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, id := range ids {
		d.expectedOrphans[id]++
	}
}

// CancelOrphan decrements the expected-orphan counter for id.
// Returns true if a counter entry was consumed, false if none existed.
// Used by call sites to undo an ExpectOrphan registration when the POST fails.
func (d *CommandDispatcher) CancelOrphan(id ObservationID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.expectedOrphans[id] > 0 {
		d.expectedOrphans[id]--
		if d.expectedOrphans[id] == 0 {
			delete(d.expectedOrphans, id)
		}
		return true
	}
	return false
}

// Send posts to uri with data, parses the Easee-specific response body, and
// if the response is asynchronous (HTTP 202), waits for the matching SignalR
// CommandResponse.
//
// Returns nil on success (both synchronous HTTP 200 and confirmed async HTTP 202,
// including noops where Ticks == 0). Returns an error on HTTP failure, decode
// failure, command rejection, or timeout.
func (d *CommandDispatcher) Send(uri string, data any) error {
	resp, err := d.helper.Post(uri, request.JSONContent, request.MarshalJSON(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	// Any status other than 200 or 202 is unexpected — return an error.
	// Note: http.Client.Post only errors on transport failures (DNS, TLS, etc.),
	// not on HTTP error responses, so this guard is the actual defense against
	// 4xx/5xx responses from the Easee API.
	if resp.StatusCode != 202 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// HTTP 202: parse the response body to get the command correlation info.
	var cmd RestCommandResponse
	if strings.Contains(uri, "/commands/") {
		// Command endpoints return a single object.
		if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil {
			return err
		}
	} else {
		// Settings endpoints return an array; take index 0 if present.
		var cmdArr []RestCommandResponse
		if err := json.NewDecoder(resp.Body).Decode(&cmdArr); err != nil {
			return err
		}
		if len(cmdArr) != 0 {
			cmd = cmdArr[0]
			for _, extra := range cmdArr[1:] {
				d.log.TRACE.Printf("ignoring additional CommandResponse in settings reply: %+v", extra)
			}
		}
	}

	if cmd.Ticks == 0 {
		// Noop: the API indicates no state change was needed.
		return nil
	}

	// Create a buffered channel (capacity 1) so Dispatch never blocks even if
	// Send has already returned due to timeout.
	ch := make(chan SignalRCommandResponse, 1)

	d.mu.Lock()
	d.pendingTicks[cmd.Ticks] = ch
	// Note: if two concurrent Send calls share the same ObservationID, the
	// second would overwrite the first's pendingByID entry. In practice this
	// cannot occur because the loadpoint serializes Enable/MaxCurrent calls.
	d.pendingByID[ObservationID(cmd.CommandId)] = ch
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.pendingTicks, cmd.Ticks)
		delete(d.pendingByID, ObservationID(cmd.CommandId))
		d.mu.Unlock()
	}()

	timer := time.NewTimer(d.timeout)
	defer timer.Stop()

	select {
	case res := <-ch:
		if !res.WasAccepted {
			return fmt.Errorf("command rejected: %d", res.Ticks)
		}
		return nil
	case <-timer.C:
		return api.ErrTimeout
	}
}
