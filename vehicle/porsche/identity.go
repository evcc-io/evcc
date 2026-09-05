package porsche

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("porsche", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			User, Password string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		if cc.User == "" || cc.Password == "" {
			return nil, api.ErrMissingCredentials
		}

		return NewIdentity(util.NewLogger("porsche"), cc.User, cc.Password)
	})
}

var (
	mu         sync.Mutex
	identities = make(map[string]*Identity)
)

// Identity is the Porsche Connect auth provider for one account. The login
// runs server-side with the stored credentials (see login.go), the token is
// persisted in the settings database.
type Identity struct {
	mu       sync.Mutex
	log      *util.Logger
	oc       *oauth2.Config
	ctx      context.Context
	subject  string
	user     string
	password string
	token    *oauth2.Token
	onlineC  chan<- bool

	loginMu sync.Mutex
	pending *LoginSession // login waiting for a captcha answer
}

var (
	_ api.AuthProvider   = (*Identity)(nil)
	_ api.AuthChallenger = (*Identity)(nil)
	_ oauth2.TokenSource = (*Identity)(nil)
)

// NewIdentity returns the identity for the account, registering it with the
// provider-auth handler on first creation. Changed credentials are picked up.
func NewIdentity(log *util.Logger, user, password string) (*Identity, error) {
	mu.Lock()
	defer mu.Unlock()

	log.Redact(user, password)

	subject := "porsche." + strings.ToLower(user)

	if o, ok := identities[subject]; ok {
		o.loginMu.Lock()
		o.password = password
		o.loginMu.Unlock()
		return o, nil
	}

	// inject X-Client-ID on all token-endpoint calls (exchange + refresh).
	// the context outlives the caller, hence it must not be request-scoped
	client := request.NewClient(log)
	client.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{"X-Client-ID": XClientID}),
		Base:      client.Transport,
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	o := &Identity{
		log:      log,
		oc:       Oauth2Config(),
		ctx:      ctx,
		subject:  subject,
		user:     user,
		password: password,
	}

	if settings.Exists(subject) {
		var token oauth2.Token
		if err := settings.Json(subject, &token); err != nil {
			return nil, err
		}
		if token.RefreshToken != "" {
			o.token = &token
			log.Redact(token.AccessToken, token.RefreshToken)
		}
	}

	onlineC, err := providerauth.Register(subject, o)
	if err != nil {
		return nil, err
	}
	o.onlineC = onlineC
	o.setOnline(o.token.Valid())

	identities[subject] = o
	return o, nil
}

// Token implements oauth2.TokenSource.
func (o *Identity) Token() (*oauth2.Token, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.token.Valid() {
		return o.token, nil
	}
	if o.token == nil || o.token.RefreshToken == "" {
		return nil, api.LoginRequiredError(o.subject)
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

// Login implements api.AuthProvider. Porsche uses the challenge login instead.
func (o *Identity) Login(string) (string, *oauth2.DeviceAuthResponse, error) {
	return "", nil, api.ErrNotAvailable
}

// HandleCallback implements api.AuthProvider. Porsche uses the challenge login instead.
func (o *Identity) HandleCallback(url.Values) error {
	return api.ErrNotAvailable
}

// Logout implements api.AuthProvider.
func (o *Identity) Logout() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.token = nil
	if settings.Exists(o.subject) {
		if err := settings.Delete(o.subject); err != nil {
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
	return "Porsche (" + o.user + ")"
}

// update stores the token and signals online status. Caller must hold o.mu.
func (o *Identity) update(token *oauth2.Token) {
	o.token = token
	o.log.Redact(token.AccessToken, token.RefreshToken)
	o.persist(token)
	o.setOnline(token.Valid())
}

func (o *Identity) persist(token *oauth2.Token) {
	if err := settings.SetJson(o.subject, token); err != nil {
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
