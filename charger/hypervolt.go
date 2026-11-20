package charger

// Hypervolt charger via the Hypervolt cloud API.
//
// Protocol as implemented by the Home Assistant integration
// https://github.com/gndean/home-assistant-hypervolt-charger:
// OAuth password login against Hypervolt's Keycloak, then a JSON-RPC
// websocket at /ws/charger/<id>/sync for state and control.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	hypervoltTokenURL  = "https://kc.prod.hypervolt.co.uk/realms/retail-customers/protocol/openid-connect/token"
	hypervoltAPI       = "https://api.hypervolt.co.uk"
	hypervoltSyncURI   = "wss://api.hypervolt.co.uk/ws/charger/%s/sync"
	hypervoltOrigin    = "https://hypervolt.co.uk"
	hypervoltClientID  = "home-assistant"
	hypervoltStaleness = 2 * time.Minute  // tolerated disconnect before data is considered stale
	hypervoltReconnect = 10 * time.Second // delay between reconnect attempts
	hypervoltPing      = time.Minute      // websocket keepalive
	hypervoltSettle    = 3 * time.Second  // wait for an initial session push after connecting
)

// Hypervolt charger implementation
type Hypervolt struct {
	log     *util.Logger
	ts      oauth2.TokenSource
	id      string
	timeout time.Duration
	cancel  context.CancelFunc

	mu           sync.RWMutex
	conn         *websocket.Conn // set only while authenticated
	disconnected time.Time       // zero while connected
	pending      map[string]chan error
	pilot        string // A, B, C
	charging     *bool
	released     *bool
	current      int64   // max current in mA
	power        float64 // W
	energy       float64 // Wh, current session
	session      bool    // session data received since connect

	writeMu sync.Mutex
	ready   chan struct{}
	once    sync.Once
	initErr chan error
}

func init() {
	registry.AddCtx("hypervolt", NewHypervoltFromConfig)
}

// NewHypervoltFromConfig creates a Hypervolt charger from generic config
func NewHypervoltFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		User, Password, ID string
		ClientID           string
		Timeout            time.Duration
	}{
		ClientID: hypervoltClientID,
		Timeout:  request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewHypervolt(ctx, cc.User, cc.Password, cc.ID, cc.ClientID, cc.Timeout)
}

// NewHypervolt creates a Hypervolt charger
func NewHypervolt(ctx context.Context, user, password, id, clientID string, timeout time.Duration) (*Hypervolt, error) {
	log := util.NewLogger("hypervolt").Redact(user, password)

	// token requests use an unlogged client so token responses never reach the trace log
	tctx := context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: timeout})

	oc := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{TokenURL: hypervoltTokenURL, AuthStyle: oauth2.AuthStyleInParams},
		Scopes:   []string{"openid", "profile", "email", "offline_access"},
	}

	pts := &hypervoltTokenSource{ctx: tctx, oc: oc, user: user, password: password}
	token, err := pts.Token()
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	log.Redact(token.AccessToken, token.RefreshToken)

	wb := &Hypervolt{
		log:     log,
		ts:      oauth2.ReuseTokenSource(token, oauth.Redacted(log, pts)),
		id:      id,
		timeout: timeout,
		pending: make(map[string]chan error),
		ready:   make(chan struct{}),
		initErr: make(chan error, 1),
	}

	if wb.id == "" {
		if wb.id, err = wb.chargerID(timeout); err != nil {
			return nil, err
		}
	}

	// the client runs for the charger's lifetime; cancel stops it if startup fails
	var rctx context.Context
	rctx, wb.cancel = context.WithCancel(ctx)
	go wb.run(rctx)

	select {
	case <-wb.ready:
		return wb, nil
	case err = <-wb.initErr:
	case <-time.After(timeout):
		err = api.ErrTimeout
	case <-ctx.Done():
		err = ctx.Err()
	}

	wb.cancel()
	return nil, err
}

// hypervoltTokenSource refreshes the token and falls back to a password login
type hypervoltTokenSource struct {
	ctx            context.Context
	oc             *oauth2.Config
	user, password string

	mu    sync.Mutex
	token *oauth2.Token
}

func (s *hypervoltTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != nil && s.token.RefreshToken != "" {
		if token, err := s.oc.TokenSource(s.ctx, s.token).Token(); err == nil {
			s.token = token
			return token, nil
		}
	}

	token, err := s.oc.PasswordCredentialsToken(s.ctx, s.user, s.password)
	if err == nil {
		s.token = token
	}

	return token, err
}

func (wb *Hypervolt) chargerID(timeout time.Duration) (string, error) {
	// unlogged transport: the account response contains personal data (name, email, picture)
	helper := request.NewHelper(wb.log)
	helper.Client = &http.Client{
		Timeout:   timeout,
		Transport: &oauth2.Transport{Source: wb.ts, Base: http.DefaultTransport},
	}

	var res struct {
		Chargers []struct {
			ID json.Number `json:"id"`
		} `json:"chargers"`
	}

	if err := helper.GetJSON(hypervoltAPI+"/users/me?includes=chargers", &res); err != nil {
		return "", err
	}

	switch len(res.Chargers) {
	case 0:
		return "", errors.New("no chargers found")
	case 1:
		return res.Chargers[0].ID.String(), nil
	default:
		ids := make([]string, 0, len(res.Chargers))
		for _, c := range res.Chargers {
			ids = append(ids, c.ID.String())
		}
		return "", fmt.Errorf("multiple chargers found, configure id: %v", ids)
	}
}

func (wb *Hypervolt) run(ctx context.Context) {
	for {
		err := wb.connect(ctx)

		wb.mu.Lock()
		wb.conn = nil
		if wb.disconnected.IsZero() {
			wb.disconnected = time.Now()
		}
		wb.failPending(errors.New("connection closed"))
		wb.mu.Unlock()

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			// report the first failure to the constructor, log the rest
			select {
			case wb.initErr <- err:
			default:
			}
			wb.log.ERROR.Println(err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(hypervoltReconnect):
		}
	}
}

func (wb *Hypervolt) connect(ctx context.Context) error {
	token, err := wb.ts.Token()
	if err != nil {
		return fmt.Errorf("token: %w", err)
	}

	dctx, cancel := context.WithTimeout(ctx, wb.timeout)
	defer cancel()

	// Authenticate via the login message, not an Authorization header: with header
	// auth the server rejects the login ("login not allowed") and does not push
	// get.session/get.pilot_status notifications.
	conn, _, err := websocket.Dial(dctx, fmt.Sprintf(hypervoltSyncURI, wb.id), &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {hypervoltOrigin}},
	})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	conn.SetReadLimit(1 << 20)
	defer conn.CloseNow()

	cctx, ccancel := context.WithCancel(ctx)
	defer ccancel()

	wb.mu.Lock()
	wb.session = false
	wb.mu.Unlock()

	// login on the private connection; it is published for commands only once authenticated
	_, loginDone, err := wb.write(cctx, conn, "login", map[string]any{"token": token.AccessToken, "version": 3})
	if err != nil {
		return err
	}

	go wb.keepalive(cctx, conn)

	for {
		_, b, err := conn.Read(cctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		if err := wb.handle(b); err != nil {
			return err
		}

		if loginDone != nil {
			select {
			case err := <-loginDone:
				loginDone = nil
				if err != nil {
					return fmt.Errorf("login: %w", err)
				}
				if err := wb.authenticated(cctx, conn); err != nil {
					return err
				}
			default:
			}
		}
	}
}

// authenticated publishes the connection and requests the initial state
func (wb *Hypervolt) authenticated(ctx context.Context, conn *websocket.Conn) error {
	wb.mu.Lock()
	wb.conn = conn
	wb.disconnected = time.Time{}
	wb.mu.Unlock()

	_, done, err := wb.write(ctx, conn, "sync.snapshot", nil)
	if err != nil {
		return err
	}

	// a rejected snapshot aborts startup with the server's error and drops the connection
	go func() {
		select {
		case err := <-done:
			if err != nil {
				err = fmt.Errorf("snapshot: %w", err)
				select {
				case wb.initErr <- err:
				default:
				}
				wb.log.ERROR.Println(err)
				conn.CloseNow()
			}
		case <-ctx.Done():
		}
	}()

	// signal readiness once the initial session push has arrived, or after a short grace period
	time.AfterFunc(hypervoltSettle, wb.signalReady)

	return nil
}

func (wb *Hypervolt) keepalive(ctx context.Context, conn *websocket.Conn) {
	for tick := time.Tick(hypervoltPing); ; {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			pctx, cancel := context.WithTimeout(ctx, wb.timeout)
			err := conn.Ping(pctx)
			cancel()
			if err != nil {
				wb.log.DEBUG.Println("ping:", err)
				conn.CloseNow()
				return
			}
		}
	}
}

// write sends a request on conn and registers it for its reply
func (wb *Hypervolt) write(ctx context.Context, conn *websocket.Conn, method string, params any) (string, chan error, error) {
	id := strconv.FormatInt(time.Now().UnixNano(), 10)

	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return "", nil, err
	}

	done := make(chan error, 1)

	wb.mu.Lock()
	wb.pending[id] = done
	wb.mu.Unlock()

	wb.log.TRACE.Printf("send: %s", method)

	wb.writeMu.Lock()
	err = conn.Write(ctx, websocket.MessageText, b)
	wb.writeMu.Unlock()

	if err != nil {
		wb.mu.Lock()
		delete(wb.pending, id)
		wb.mu.Unlock()
		return "", nil, err
	}

	return id, done, nil
}

// call sends a command on the authenticated connection and waits for the server's reply
func (wb *Hypervolt) call(method string, params any) error {
	wb.mu.RLock()
	conn := wb.conn
	wb.mu.RUnlock()

	if conn == nil {
		return errors.New("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), wb.timeout)
	defer cancel()

	id, done, err := wb.write(ctx, conn, method, params)
	if err != nil {
		return err
	}

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		wb.mu.Lock()
		delete(wb.pending, id)
		wb.mu.Unlock()
		return api.ErrTimeout
	}
}

// failPending completes all outstanding requests with err; requires wb.mu
func (wb *Hypervolt) failPending(err error) {
	for id, done := range wb.pending {
		done <- err
		delete(wb.pending, id)
	}
}

func (wb *Hypervolt) signalReady() {
	wb.mu.RLock()
	ok := wb.released != nil
	wb.mu.RUnlock()

	if ok {
		wb.once.Do(func() { close(wb.ready) })
	}
}

type hypervoltMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Params json.RawMessage `json:"params"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// handle processes a single websocket message
func (wb *Hypervolt) handle(b []byte) error {
	var msg hypervoltMessage
	if err := json.Unmarshal(b, &msg); err != nil {
		return nil // ignore unknown messages
	}

	// reply to one of our requests
	if len(msg.ID) > 0 && msg.Method == "" {
		var id string
		if err := json.Unmarshal(msg.ID, &id); err != nil {
			id = string(msg.ID)
		}

		wb.mu.Lock()
		done, ok := wb.pending[id]
		delete(wb.pending, id)
		wb.mu.Unlock()

		var err error
		switch {
		case msg.Error != nil:
			err = fmt.Errorf("%s (%d)", msg.Error.Message, msg.Error.Code)
			wb.log.DEBUG.Printf("recv: error %v", err)
		case isHypervoltLoginResult(msg.Result) && !hypervoltAuthenticated(msg.Result):
			err = errors.New("not authenticated")
		default:
			wb.log.TRACE.Printf("recv: reply %s", id)
			wb.applySync(msg.Result)
		}

		if ok {
			done <- err
		}

		return nil
	}

	wb.log.TRACE.Printf("recv: %s", msg.Method)

	switch msg.Method {
	case "sync.snapshot", "sync.apply":
		wb.applySync(msg.Params)

	case "get.session":
		wb.applySession(msg.Params)

	case "get.pilot_status":
		var res struct {
			PilotStatus string `json:"pilot_status"`
		}
		if json.Unmarshal(msg.Params, &res) == nil && res.PilotStatus != "" {
			wb.mu.Lock()
			wb.pilot = res.PilotStatus
			wb.mu.Unlock()
		}
	}

	return nil
}

func isHypervoltLoginResult(data json.RawMessage) bool {
	var res map[string]json.RawMessage
	if json.Unmarshal(data, &res) != nil {
		return false
	}
	_, ok := res["authenticated"]
	return ok
}

func hypervoltAuthenticated(data json.RawMessage) bool {
	var res struct {
		Authenticated bool `json:"authenticated"`
	}
	return json.Unmarshal(data, &res) == nil && res.Authenticated
}

// applySync handles sync payloads which are either an object or a list of single-key objects
func (wb *Hypervolt) applySync(data json.RawMessage) {
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(data, &item); err != nil {
			return
		}
		items = append(items, item)
	}

	wb.mu.Lock()
	for _, item := range items {
		if v, ok := item["release_state"]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				released := s == "released"
				wb.released = &released
			}
		}
		if v, ok := item["max_current"]; ok {
			_ = json.Unmarshal(v, &wb.current)
		}
	}
	session := wb.session
	wb.mu.Unlock()

	// session already seen: release state was the missing piece
	if session {
		wb.signalReady()
	}
}

func (wb *Hypervolt) applySession(data json.RawMessage) {
	var res struct {
		Charging  *bool    `json:"charging"`
		WattHours *float64 `json:"watt_hours"`
		EvPower   *float64 `json:"ev_power"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return
	}

	wb.mu.Lock()
	if res.Charging != nil {
		wb.charging = res.Charging
	}
	if res.WattHours != nil {
		wb.energy = *res.WattHours
	}
	if res.EvPower != nil {
		wb.power = *res.EvPower
	}
	first := !wb.session
	wb.session = true
	wb.mu.Unlock()

	// the initial session reading makes the startup energy baseline correct
	if first {
		wb.signalReady()
	}
}

// stale returns an error only if the connection has been down for too long.
// An idle charger sends no updates, so its last state stays valid while connected.
func (wb *Hypervolt) stale() error {
	if !wb.disconnected.IsZero() && time.Since(wb.disconnected) > hypervoltStaleness {
		return api.ErrTimeout
	}
	return nil
}

// Status implements the api.Charger interface
func (wb *Hypervolt) Status() (api.ChargeStatus, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	if err := wb.stale(); err != nil {
		return api.StatusNone, err
	}

	switch wb.pilot {
	case "A":
		return api.StatusA, nil
	case "B":
		return api.StatusB, nil
	case "C":
		return api.StatusC, nil
	}

	if wb.charging != nil && *wb.charging {
		return api.StatusC, nil
	}

	return api.StatusNone, errors.New("unknown status")
}

// Enabled implements the api.Charger interface
func (wb *Hypervolt) Enabled() (bool, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	if wb.released == nil {
		return false, errors.New("unknown enabled state")
	}

	return !*wb.released, wb.stale()
}

// Enable implements the api.Charger interface
func (wb *Hypervolt) Enable(enable bool) error {
	return wb.call("sync.apply", map[string]any{"release": !enable})
}

// MaxCurrent implements the api.Charger interface
func (wb *Hypervolt) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Hypervolt)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Hypervolt) MaxCurrentMillis(current float64) error {
	return wb.call("sync.apply", map[string]any{"max_current": int64(current * 1e3)})
}

var _ api.CurrentGetter = (*Hypervolt)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *Hypervolt) GetMaxCurrent() (float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	return float64(wb.current) / 1e3, wb.stale()
}

var _ api.Meter = (*Hypervolt)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Hypervolt) CurrentPower() (float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	return wb.power, wb.stale()
}

var _ api.ChargeRater = (*Hypervolt)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Hypervolt) ChargedEnergy() (float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	return wb.energy / 1e3, wb.stale()
}
