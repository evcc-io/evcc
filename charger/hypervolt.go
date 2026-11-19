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
	hypervoltStaleness = 5 * time.Minute
	hypervoltReconnect = 10 * time.Second
)

// Hypervolt charger implementation
type Hypervolt struct {
	log *util.Logger
	ts  oauth2.TokenSource
	id  string

	mu       sync.RWMutex
	conn     *websocket.Conn
	pending  map[string]string // message id -> method
	updated  time.Time
	pilot    string // A, B, C
	charging *bool
	released *bool
	current  int64   // max current in mA
	power    float64 // W
	energy   float64 // Wh, current session

	writeMu sync.Mutex
	ready   chan struct{}
	once    sync.Once
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

	client := request.NewClient(log)
	client.Timeout = timeout
	tctx := context.WithValue(ctx, oauth2.HTTPClient, client)

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

	wb := &Hypervolt{
		log:     log,
		ts:      oauth2.ReuseTokenSource(token, oauth.Redacted(log, pts)),
		id:      id,
		pending: make(map[string]string),
		ready:   make(chan struct{}),
	}

	if wb.id == "" {
		if wb.id, err = wb.chargerID(client); err != nil {
			return nil, err
		}
	}

	go wb.run(ctx)

	select {
	case <-wb.ready:
	case <-time.After(timeout):
		return nil, api.ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return wb, nil
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

func (wb *Hypervolt) chargerID(client *http.Client) (string, error) {
	helper := request.NewHelper(wb.log)
	helper.Client = &http.Client{
		Timeout:   client.Timeout,
		Transport: &oauth2.Transport{Source: wb.ts, Base: client.Transport},
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
	for ctx.Err() == nil {
		if err := wb.connect(ctx); err != nil && ctx.Err() == nil {
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

	dctx, cancel := context.WithTimeout(ctx, request.Timeout)
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

	wb.mu.Lock()
	wb.conn = conn
	wb.pending = make(map[string]string)
	wb.mu.Unlock()

	defer func() {
		wb.mu.Lock()
		wb.conn = nil
		wb.mu.Unlock()
	}()

	if err := wb.send(ctx, "login", map[string]any{"token": token.AccessToken, "version": 3}); err != nil {
		return err
	}

	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		wb.log.TRACE.Printf("recv: %s", b)

		if err := wb.handle(ctx, b); err != nil {
			return err
		}
	}
}

func (wb *Hypervolt) send(ctx context.Context, method string, params any) error {
	id := strconv.FormatInt(time.Now().UnixNano(), 10)

	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	wb.writeMu.Lock()
	defer wb.writeMu.Unlock()

	wb.mu.Lock()
	conn := wb.conn
	if conn != nil {
		wb.pending[id] = method
	}
	wb.mu.Unlock()

	if conn == nil {
		return errors.New("not connected")
	}

	if method != "login" {
		wb.log.TRACE.Printf("send: %s", b)
	}

	return conn.Write(ctx, websocket.MessageText, b)
}

type hypervoltMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Params json.RawMessage `json:"params"`
	Error  json.RawMessage `json:"error"`
}

// handle processes a single websocket message
func (wb *Hypervolt) handle(ctx context.Context, b []byte) error {
	var msg hypervoltMessage
	if err := json.Unmarshal(b, &msg); err != nil {
		return nil // ignore unknown messages
	}

	method, data := msg.Method, msg.Params
	if msg.Result != nil {
		data = msg.Result

		var id string
		if err := json.Unmarshal(msg.ID, &id); err != nil {
			id = string(msg.ID)
		}

		wb.mu.Lock()
		if m, ok := wb.pending[id]; ok {
			method = m
			delete(wb.pending, id)
		}
		wb.mu.Unlock()
	}

	if len(msg.Error) > 0 {
		wb.log.WARN.Printf("error: %s", msg.Error)
		return nil
	}

	switch method {
	case "login":
		var res struct {
			Authenticated bool `json:"authenticated"`
		}
		if err := json.Unmarshal(data, &res); err != nil || !res.Authenticated {
			return errors.New("websocket login failed")
		}
		return wb.send(ctx, "sync.snapshot", nil)

	case "sync.snapshot", "sync.apply":
		wb.applySync(data)

	case "get.session":
		wb.applySession(data)

	case "get.pilot_status":
		var res struct {
			PilotStatus string `json:"pilot_status"`
		}
		if json.Unmarshal(data, &res) == nil && res.PilotStatus != "" {
			wb.mu.Lock()
			wb.pilot = res.PilotStatus
			wb.updated = time.Now()
			wb.mu.Unlock()
		}
	}

	return nil
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
	defer wb.mu.Unlock()

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

	wb.updated = time.Now()

	if wb.released != nil {
		wb.once.Do(func() { close(wb.ready) })
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
	defer wb.mu.Unlock()

	if res.Charging != nil {
		wb.charging = res.Charging
	}
	if res.WattHours != nil {
		wb.energy = *res.WattHours
	}
	if res.EvPower != nil {
		wb.power = *res.EvPower
	}
	wb.updated = time.Now()
}

func (wb *Hypervolt) stale() error {
	if time.Since(wb.updated) > hypervoltStaleness {
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
	return wb.send(context.Background(), "sync.apply", map[string]any{"release": !enable})
}

// MaxCurrent implements the api.Charger interface
func (wb *Hypervolt) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Hypervolt)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Hypervolt) MaxCurrentMillis(current float64) error {
	return wb.send(context.Background(), "sync.apply", map[string]any{"max_current": int64(current * 1e3)})
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
