package porsche

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// subject is the stable provider id; a single Porsche account serves all of the
// user's Porsche vehicles, so the identity (and its token) is a singleton.
const subject = "porsche"

var (
	mu       sync.Mutex
	identity *Identity
)

// Identity is the Porsche Connect auth provider. It implements both
// api.AuthProvider (browser login via evcc's providerauth) and
// oauth2.TokenSource (used by the API client), persisting the token in the
// settings database.
type Identity struct {
	mu      sync.Mutex
	log     *util.Logger
	oc      *oauth2.Config
	ctx     context.Context
	token   *oauth2.Token
	onlineC chan<- bool

	loginMu sync.Mutex
	pending *LoginSession // in-progress interactive (captcha) login
}

var (
	_ api.AuthProvider   = (*Identity)(nil)
	_ oauth2.TokenSource = (*Identity)(nil)
)

// NewIdentity returns the (singleton) Porsche identity, registering it with
// evcc's provider-auth handler on first creation. A non-nil seed token (e.g.
// from `evcc token` in the config) is used when the database has none yet.
func NewIdentity(ctx context.Context, log *util.Logger, seed *oauth2.Token) (*Identity, error) {
	mu.Lock()
	defer mu.Unlock()

	if identity != nil {
		return identity, nil
	}

	// inject X-Client-ID on all token-endpoint calls (exchange + refresh)
	client := request.NewClient(log)
	client.Transport = &headerRoundTripper{base: client.Transport}
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, client)

	o := &Identity{
		log: log,
		oc:  Oauth2Config(),
		ctx: authCtx,
	}

	// load persisted token, else fall back to the seed token
	var token oauth2.Token
	if settings.Exists(subject) {
		if err := settings.Json(subject, &token); err != nil {
			return nil, err
		}
	} else if seed != nil && seed.RefreshToken != "" {
		token = *seed
	}
	if token.RefreshToken != "" {
		o.token = &token
		if !settings.Exists(subject) {
			o.persist(&token)
		}
	}

	onlineC, err := providerauth.Register(subject, o)
	if err != nil {
		return nil, err
	}
	o.onlineC = onlineC
	o.setOnline(o.token.Valid())

	identity = o
	return o, nil
}

// Token implements oauth2.TokenSource.
func (o *Identity) Token() (*oauth2.Token, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.token == nil {
		return nil, api.LoginRequiredError(subject)
	}
	if o.token.Valid() {
		return o.token, nil
	}
	if o.token.RefreshToken == "" {
		return nil, api.LoginRequiredError(subject)
	}

	// oauth2 handles the refresh via o.ctx's client, which injects the required
	// X-Client-ID header (same path as the initial code exchange)
	token, err := o.oc.TokenSource(o.ctx, &oauth2.Token{RefreshToken: o.token.RefreshToken}).Token()
	if err != nil {
		return nil, err
	}

	o.update(token)
	return token, nil
}

// Login implements api.AuthProvider. No PKCE (the Porsche Auth0 client does not
// use it); audience scopes the access token to the Porsche API.
func (o *Identity) Login(state string) (string, *oauth2.DeviceAuthResponse, error) {
	return o.oc.AuthCodeURL(state, oauth2.SetAuthURLParam("audience", Audience)), nil, nil
}

// HandleCallback implements api.AuthProvider.
func (o *Identity) HandleCallback(params url.Values) error {
	if e := params.Get("error"); e != "" {
		return fmt.Errorf("%s: %s", e, params.Get("error_description"))
	}

	code := params.Get("code")
	if code == "" {
		return errors.New("missing authorization code")
	}

	token, err := o.oc.Exchange(o.ctx, code)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.update(token)
	return nil
}

// Logout implements api.AuthProvider.
func (o *Identity) Logout() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.token = nil
	if settings.Exists(subject) {
		if err := settings.Delete(subject); err != nil {
			o.log.ERROR.Println(err)
		}
	}
	o.setOnline(false)
	return nil
}

// Authenticated implements api.AuthProvider.
func (o *Identity) Authenticated() bool {
	token, err := o.Token()
	return err == nil && token.Valid()
}

// DisplayName implements api.AuthProvider.
func (o *Identity) DisplayName() string {
	return "Porsche"
}

// update stores the token and signals online status. Caller must hold o.mu.
func (o *Identity) update(token *oauth2.Token) {
	if token.RefreshToken == "" && o.token != nil {
		token.RefreshToken = o.token.RefreshToken
	}
	o.token = token
	o.persist(token)
	o.setOnline(token.Valid())
}

func (o *Identity) persist(token *oauth2.Token) {
	if err := settings.SetJson(subject, token); err != nil {
		o.log.ERROR.Printf("saving token: %v", err)
		return
	}
	// flush immediately so the token survives a restart right after login/refresh
	if err := settings.Persist(); err != nil {
		o.log.ERROR.Printf("persisting token: %v", err)
	}
}

// setOnline signals the auth handler without blocking (a blocking send under
// o.mu would deadlock via Authenticated()->Token()).
func (o *Identity) setOnline(online bool) {
	select {
	case o.onlineC <- online:
	default:
	}
}

// headerRoundTripper adds the X-Client-ID header required by the Porsche token
// endpoint to every request (used for the OAuth code exchange).
type headerRoundTripper struct {
	base http.RoundTripper
}

func (t *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Client-ID", XClientID)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
