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

	if tickOk {
		chTick <- res
		return
	}

	if idOk {
		chID <- res
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
		return true
	}
	return false
}

// Send posts to uri with data, parses the Easee-specific response body, and
// if the response is asynchronous (HTTP 202), waits for the matching SignalR
// CommandResponse. Implemented in Task 4.
func (d *CommandDispatcher) Send(uri string, data any) error {
	panic("not implemented")
}

// suppress unused import errors during incremental development
var (
	_ = fmt.Sprintf
	_ = json.NewDecoder
	_ = strings.Contains
	_ = api.ErrTimeout
)
