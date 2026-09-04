package mercedes

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	pb "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	// wsReadTimeout bounds a single read. The backend sends a full update right
	// after connecting and keeps a keep-alive going; a read blocking this long
	// means the connection is dead and we reconnect.
	wsReadTimeout = 60 * time.Second

	// wsPingInterval keeps the connection alive between VSU pushes.
	wsPingInterval = 30 * time.Second

	// wsReconnectDelay is the minimum delay between connection attempts. The
	// backend pushes a full update on connect, so this is also the effective
	// poll interval; keep it comfortably above zero to avoid 429 account blocks
	// (mbapi2020 uses a comparable duty cycle).
	wsReconnectDelay = 60 * time.Second

	// wsMaxMessageBytes bounds a single websocket frame. VSU full updates carry
	// ~250 attributes per vehicle; 8 MiB is comfortably above that.
	wsMaxMessageBytes = 8 << 20
)

// wsState is one vehicle's cached VSU together with the time it was last
// updated, used by the provider's freshness guard.
type wsState struct {
	vsu     *pb.VehicleStatusUpdate
	updated time.Time
}

// Websocket maintains a single VSU push connection per Mercedes account and
// caches the latest VehicleStatusUpdate per VIN. It is long-lived: there is one
// instance per account (see the registry in helper.go), started once and kept
// running for the process lifetime, mirroring evcc's websocket chargers.
type Websocket struct {
	mu        sync.RWMutex
	log       *util.Logger
	identity  *Identity
	region    string
	sessionID string
	states    map[string]*wsState                 // keyed by VIN
	commands  map[string]*pb.AppTwinCommandStatus // latest command result, keyed by VIN

	// conn is the current connection, guarded by mu; used to send acks from the
	// read loop.
	connMu sync.Mutex
	conn   *websocket.Conn
}

var (
	wsMu       sync.Mutex
	websockets = make(map[string]*Websocket)

	errNotConnected = errors.New("websocket not connected")
)

// NewWebsocket returns the shared Websocket for the given account, creating and
// starting it on first use. Subsequent vehicles on the same account reuse it.
func NewWebsocket(log *util.Logger, identity *Identity) *Websocket {
	wsMu.Lock()
	defer wsMu.Unlock()

	key := identity.account + "-" + identity.region
	if ws, ok := websockets[key]; ok {
		return ws
	}

	ws := &Websocket{
		log:       log,
		identity:  identity,
		region:    identity.region,
		sessionID: uuid.New().String(),
		states:    make(map[string]*wsState),
		commands:  make(map[string]*pb.AppTwinCommandStatus),
	}

	// Process-lifetime goroutine. api.Vehicle has no Close(); the account
	// registry guarantees at most one manager per account.
	go ws.run(context.Background())

	websockets[key] = ws
	return ws
}

// Status returns the latest cached VSU for the given VIN and the time it was
// received. ok is false until the first update for that VIN arrives.
func (ws *Websocket) Status(vin string) (*pb.VehicleStatusUpdate, time.Time, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	s, ok := ws.states[vin]
	if !ok || s.vsu == nil {
		return nil, time.Time{}, false
	}
	return s.vsu, s.updated, true
}

// commandStatus returns the latest command result whose request id matches
// reqID (the id returned by sendCommand). ok is false until that command's
// status push arrives. Results are cached latest-per-VIN, so a newer command to
// the same vehicle replaces an older one.
func (ws *Websocket) commandStatus(reqID string) (*pb.AppTwinCommandStatus, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	for _, st := range ws.commands {
		if st.GetRequestId() == reqID {
			return st, true
		}
	}
	return nil, false
}

// sendCommand issues a vehicle CommandRequest over the websocket. It stamps a
// request id when one is not set and returns it so the caller can correlate the
// asynchronous result via commandStatus. It does not block waiting for a
// reconnect: if the connection is momentarily down it returns errNotConnected so
// the caller can retry.
func (ws *Websocket) sendCommand(ctx context.Context, req *pb.CommandRequest) (string, error) {
	if req.GetRequestId() == "" {
		req.RequestId = uuid.New().String()
	}
	return req.GetRequestId(), ws.writeMessage(ctx, &pb.ClientMessage{
		Msg: &pb.ClientMessage_CommandRequest{CommandRequest: req},
	})
}

// run maintains the connection with exponential backoff. A fresh full update
// arrives after every (re)connect, so each reconnect doubles as a poll.
//
// A minimum reconnect delay is always enforced between connection attempts
// (even successful ones): mbapi2020 documents that reconnecting too eagerly can
// trigger HTTP 429 account blocks, so the connection runs on a duty cycle
// rather than reconnecting instantly.
func (ws *Websocket) run(ctx context.Context) {
	bo := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(0),
		backoff.WithMaxInterval(2*time.Minute),
	)

	for ctx.Err() == nil {
		err := ws.connect(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			ws.log.DEBUG.Printf("websocket: %v", err)
		}

		// On errors use exponential backoff; on a clean disconnect fall back to
		// the minimum reconnect delay.
		delay := wsReconnectDelay
		if err != nil {
			if d := bo.NextBackOff(); d > delay {
				delay = d
			}
		} else {
			bo.Reset()
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// connect dials the websocket and runs the read loop until the connection
// drops. It returns the terminating error (nil on a clean close).
func (ws *Websocket) connect(ctx context.Context) error {
	tok, err := ws.identity.Token()
	if err != nil {
		return err
	}

	headers := wsheaders(tok.AccessToken, ws.sessionID, ws.region)
	opts := &websocket.DialOptions{HTTPHeader: mapToHeader(headers)}

	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	conn, _, err := websocket.Dial(dialCtx, getWebsocketUri(ws.region), opts)
	cancel()
	if err != nil {
		return err
	}
	conn.SetReadLimit(wsMaxMessageBytes)

	ws.connMu.Lock()
	ws.conn = conn
	ws.connMu.Unlock()

	defer func() {
		ws.connMu.Lock()
		ws.conn = nil
		ws.connMu.Unlock()
		_ = conn.CloseNow()
	}()

	ws.log.DEBUG.Println("websocket: connected")

	// Ping keep-alive. Liveness is proven by pings, not by data: the pinger
	// tears the connection down when a ping fails, so a stalled connection is
	// surfaced as a read error rather than being mistaken for an idle cycle.
	pingCtx, pingCancel := context.WithCancel(ctx)
	defer pingCancel()
	go ws.pinger(pingCtx, conn)

	var gotData bool
	for {
		readCtx, readCancel := context.WithTimeout(ctx, wsReadTimeout)
		typ, data, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			if idleCycleEnd(err, ctx.Err(), gotData) {
				// Clean end of a poll cycle: the backend pushed its update(s)
				// and then went quiet, and the keep-alive pings are still
				// succeeding. Return nil so run() reconnects on the steady duty
				// cycle and resets the backoff instead of treating this normal
				// condition as an error.
				return nil
			}
			return err
		}
		if typ != websocket.MessageBinary {
			continue
		}
		gotData = true

		if err := ws.handleMessage(ctx, data); err != nil {
			ws.log.DEBUG.Printf("websocket: handle: %v", err)
		}
	}
}

// idleCycleEnd reports whether a read error is the benign end of a poll cycle:
// our own read deadline fired (context.DeadlineExceeded) rather than a
// parent-context cancellation (parentErr != nil), and the connection had already
// delivered at least one message. A healthy connection to a parked vehicle
// always ends this way, so it must not be logged as an error or drive the
// reconnect backoff upward.
//
// The gotData guard keeps a genuine fault visible: a connection that times out
// having delivered nothing is abnormal (the backend pushes a full update on
// connect) and is reported as an error so it is logged and backed off. A
// connection that dies mid-life is caught by the pinger, which closes it and
// makes Read return a non-deadline error.
func idleCycleEnd(err, parentErr error, gotData bool) bool {
	return gotData && parentErr == nil && errors.Is(err, context.DeadlineExceeded)
}

func (ws *Websocket) pinger(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				// A failed ping means the connection is dead. Close it so the
				// read loop unblocks immediately with a non-deadline error and
				// reconnects, instead of waiting out the full read timeout and
				// mistaking the dead connection for a clean idle cycle. Skip the
				// close on parent-context teardown, where connect() already
				// closes the connection.
				if ctx.Err() == nil {
					_ = conn.CloseNow()
				}
				return
			}
		}
	}
}

// handleMessage decodes a PushMessage and acknowledges it. Every
// sequence-numbered push type must be acknowledged or the backend redelivers it
// and eventually drops the connection, so both vehicle_status_updates (the data
// we cache) and apptwin_command_status_updates (results of commands we sent) are
// handled and acked. Other push types are ignored.
func (ws *Websocket) handleMessage(ctx context.Context, data []byte) error {
	var msg pb.PushMessage
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}

	var ack *pb.ClientMessage
	switch {
	case msg.GetVehicleStatusUpdates() != nil:
		vsus := msg.GetVehicleStatusUpdates()
		ws.applyVSU(vsus)
		ack = &pb.ClientMessage{Msg: &pb.ClientMessage_AcknowledgeVehicleStatusUpdates{
			AcknowledgeVehicleStatusUpdates: &pb.AcknowledgeVehicleStatusUpdates{SequenceNumber: vsus.GetSequenceNumber()},
		}}

	case msg.GetApptwinCommandStatusUpdatesByVin() != nil:
		// Result of a command we (or another session) sent. Cache it for
		// SendCommand callers and ack it: the push is sequence-numbered, and an
		// unacknowledged push is redelivered and can eventually make the backend
		// drop the connection.
		cmd := msg.GetApptwinCommandStatusUpdatesByVin()
		ws.applyCommandStatus(cmd)
		ack = &pb.ClientMessage{Msg: &pb.ClientMessage_AcknowledgeApptwinCommandStatusUpdateByVin{
			AcknowledgeApptwinCommandStatusUpdateByVin: &pb.AcknowledgeAppTwinCommandStatusUpdatesByVIN{SequenceNumber: cmd.GetSequenceNumber()},
		}}

	default:
		return nil
	}

	return ws.writeMessage(ctx, ack)
}

// writeMessage marshals a ClientMessage and writes it on the current connection.
// connMu only guards fetching the connection pointer; the blocking write happens
// outside the lock (coder/websocket serialises concurrent writes internally).
func (ws *Websocket) writeMessage(ctx context.Context, msg *pb.ClientMessage) error {
	out, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	ws.connMu.Lock()
	conn := ws.conn
	ws.connMu.Unlock()
	if conn == nil {
		return errNotConnected
	}

	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageBinary, out)
}

// applyVSU merges each vehicle's update into the cache. full_update replaces the
// entry; a partial update is merged field-wise (see mergeVSU).
func (ws *Websocket) applyVSU(vsus *pb.VehicleStatusUpdates) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for vin, upd := range vsus.GetVehicleStatusUpdates() {
		if upd == nil {
			continue
		}

		s, ok := ws.states[vin]
		if !ok {
			s = &wsState{}
			ws.states[vin] = s
		}

		if upd.GetFullUpdate() || s.vsu == nil {
			// Clone so the cache owns its copy and later merges don't mutate the
			// received message.
			s.vsu = proto.Clone(upd).(*pb.VehicleStatusUpdate)
		} else {
			// Merge into a fresh clone and swap the pointer atomically. A reader
			// that obtained the previous *VehicleStatusUpdate from Status() keeps
			// an immutable snapshot, so no lock is needed on the read side beyond
			// fetching the pointer.
			merged := proto.Clone(s.vsu).(*pb.VehicleStatusUpdate)
			mergeVSU(merged, upd)
			s.vsu = merged
		}
		s.updated = time.Now()

		ws.log.TRACE.Printf("websocket: VSU %s full=%v", vin, upd.GetFullUpdate())
	}
}

// applyCommandStatus caches the latest command result per VIN and logs its
// state/errors so a command's outcome (ENQUEUED → PROCESSING → FINISHED/FAILED)
// is observable to SendCommand callers.
func (ws *Websocket) applyCommandStatus(upd *pb.AppTwinCommandStatusUpdatesByVIN) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for vin, byPid := range upd.GetUpdatesByVin() {
		for pid, st := range byPid.GetUpdatesByPid() {
			if st == nil {
				continue
			}
			ws.commands[vin] = st

			ws.log.TRACE.Printf("websocket: command %s vin=%s pid=%d type=%s state=%s", st.GetRequestId(), vin, pid, st.GetType(), st.GetState())
			for _, e := range st.GetErrors() {
				ws.log.TRACE.Printf("websocket: command %s error code=%q message=%q", st.GetRequestId(), e.GetCode(), e.GetMessage())
			}
		}
	}
}

// mapToHeader converts a header map to http.Header for the dialer.
func mapToHeader(m map[string]string) map[string][]string {
	h := make(map[string][]string, len(m))
	for k, v := range m {
		h[k] = []string{v}
	}
	return h
}
