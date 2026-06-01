package remote

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Settings is the persisted remote access configuration.
type Settings struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url,omitempty"`
	Token     string `json:"token,omitempty"`
	TunnelURL string `json:"tunnelUrl,omitempty"`
}

// Remote manages the remote access tunnel lifecycle.
type Remote struct {
	mu          sync.Mutex
	cloudHost   string
	settings    Settings
	tunnel      *Tunnel
	httpHandler http.Handler
	log         *util.Logger
	publisher   chan<- util.Param
	lastSeen    map[string]time.Time // persisted: username → last activity
	connected   map[string]int       // in-memory: active connection count per user
}

// New creates a new Remote manager, loads persisted settings, and connects if enabled.
func New(cloudHost string, httpHandler http.Handler, valueChan chan<- util.Param) *Remote {
	r := &Remote{
		cloudHost:   cloudHost,
		httpHandler: httpHandler,
		log:         util.NewLogger("remote"),
		publisher:   valueChan,
		lastSeen:    make(map[string]time.Time),
		connected:   make(map[string]int),
	}

	// load saved settings
	_ = settings.Json(keys.Remote, &r.settings)
	_ = settings.Json(keys.RemoteLastSeen, &r.lastSeen)

	if r.settings.Enabled && r.settings.Token != "" {
		go r.connect()
	}

	shutdown.Register(r.disconnect)

	go func() {
		for range time.Tick(time.Minute) {
			r.publish()
		}
	}()

	return r
}

// Enable enables or disables remote access. When enabling for the first time,
// it registers with the cloud to obtain a URL and token.
func (r *Remote) Enable(enable bool) error {
	r.mu.Lock()
	r.settings.Enabled = enable
	r.saveSettings()
	r.mu.Unlock()

	if enable {
		// TODO why do we need a go routine for this?
		go r.connect()
	} else {
		r.disconnect()
	}

	r.publish()
	return nil
}

// Enabled returns whether remote access is enabled.
func (r *Remote) Enabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.settings.Enabled
}

func (r *Remote) connect() {
	r.mu.Lock()
	token := r.settings.Token
	r.mu.Unlock()

	if token == "" {
		if err := r.register(); err != nil {
			r.log.ERROR.Printf("registration failed: %v", err)
			return
		}
	}

	r.log.INFO.Printf("remote access via %s", r.settings.URL)

	tunnel := NewTunnel(r.settings.TunnelURL, r.settings.Token, r.httpHandler, r.Authenticate, r.TrackActivity, r.log, r.publish)

	r.mu.Lock()
	r.tunnel = tunnel
	r.mu.Unlock()

	// blocks until disconnected
	tunnel.run()
}

func (r *Remote) disconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tunnel != nil {
		r.tunnel.Close()
		r.tunnel = nil
	}
}

type registerRequest struct {
	SponsorToken string `json:"sponsorToken"`
}

type registerResponse struct {
	URL       string `json:"url"`
	Token     string `json:"token"`
	TunnelURL string `json:"tunnelUrl"`
}

// register calls the cloud registration endpoint and persists the result.
func (r *Remote) register() error {
	uri := fmt.Sprintf("https://%s/api/register", r.cloudHost)
	data := registerRequest{SponsorToken: sponsor.Token}
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	var res registerResponse

	client := request.NewHelper(r.log)
	if err := client.DoJSON(req, &res); err != nil {
		return err
	}

	r.mu.Lock()
	r.settings.URL = res.URL
	r.settings.Token = res.Token
	r.settings.TunnelURL = res.TunnelURL
	r.saveSettings()
	r.mu.Unlock()

	r.log.INFO.Printf("registered as %s", res.URL)
	return nil
}

// TrackActivity tracks remote client connections and disconnections.
func (r *Remote) TrackActivity(username string, active bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if active {
		r.lastSeen[username] = time.Now()
		r.connected[username]++
	} else if r.connected[username] > 0 {
		r.connected[username]--
	}
}

// saveSettings persists the current settings. Must be called with mu held.
func (r *Remote) saveSettings() {
	if err := settings.SetJson(keys.Remote, r.settings); err != nil {
		r.log.ERROR.Println(err)
	}
}

// ConfigStatus returns the current remote access config and status.
func (r *Remote) ConfigStatus() globalconfig.ConfigStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	connected := r.tunnel != nil && r.tunnel.IsConnected()
	loginBlocked := r.tunnel != nil && r.tunnel.LoginBlocked()

	return globalconfig.ConfigStatus{
		Config: struct {
			Enabled bool `json:"enabled"`
		}{
			Enabled: r.settings.Enabled,
		},
		Status: struct {
			Connected    bool                 `json:"connected"`
			URL          string               `json:"url,omitempty"`
			LoginBlocked bool                 `json:"loginBlocked"`
			LastSeen     map[string]time.Time `json:"lastSeen,omitempty"`
		}{
			Connected:    connected,
			URL:          r.settings.URL,
			LoginBlocked: loginBlocked,
			LastSeen:     r.lastSeen,
		},
	}
}

// publish sends the current status to the UI via the value channel.
func (r *Remote) publish() {
	if r.publisher == nil {
		return
	}

	// refresh lastSeen for open connections (auth only fires once)
	r.mu.Lock()
	now := time.Now()
	for user, count := range r.connected {
		if count > 0 {
			r.lastSeen[user] = now
		}
	}
	_ = settings.SetJson(keys.RemoteLastSeen, r.lastSeen)
	r.mu.Unlock()

	r.publisher <- util.Param{Key: keys.Remote, Val: r.ConfigStatus()}
}
