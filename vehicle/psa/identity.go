package psa

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// brandNames maps the registry name to the name shown in the ui
var brandNames = map[string]string{
	"citroen": "Citroën",
	"ds":      "DS",
	"opel":    "Opel",
	"peugeot": "Peugeot",
}

func init() {
	for brand := range brandNames {
		auth.Register(brand, func(other map[string]any) (oauth2.TokenSource, error) {
			var cc struct {
				User, Country string
			}
			if err := util.DecodeOther(other, &cc); err != nil {
				return nil, err
			}
			if cc.User == "" {
				return nil, api.ErrMissingCredentials
			}
			return NewIdentity(util.NewLogger(brand), brand, cc.User, cc.Country, nil)
		})
	}
}

// pendingLogin is an interactive login waiting for the user-supplied code
type pendingLogin struct {
	oc       *oauth2.Config
	verifier string
}

// Identity is the PSA auth provider. Login is interactive rather than a
// redirect flow: the identity provider only accepts the mobile app's private
// url scheme as redirect uri, so the user signs in in their own browser and
// pastes the authorization code back into evcc.
type Identity struct {
	mu      sync.Mutex
	log     *util.Logger
	oc      *oauth2.Config
	ctx     context.Context
	brand   string
	user    string
	country string
	subject string
	token   *oauth2.Token
	onlineC chan<- bool

	loginMu sync.Mutex
	pending *pendingLogin
}

var (
	_ api.AuthProvider   = (*Identity)(nil)
	_ api.AuthChallenger = (*Identity)(nil)
	_ oauth2.TokenSource = (*Identity)(nil)
)

// NewIdentity creates the PSA identity for brand and user, registering it with
// evcc's provider-auth handler on first creation. A non-nil seed token (e.g.
// from `evcc token` in the config) is used when the database has none yet.
func NewIdentity(log *util.Logger, brand, user, country string, seed *oauth2.Token) (*Identity, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	log.Redact(user)
	country = strings.ToLower(strings.TrimSpace(country))
	if country == "" {
		country = "de"
	}

	// reuse identity instance
	subject := "psa." + strings.ToLower(brand) + "." + strings.ToLower(user)
	if instance := getInstance(subject); instance != nil {
		instance.loginMu.Lock()
		if instance.country != country {
			instance.country = country
			instance.pending = nil
		}
		instance.loginMu.Unlock()
		return instance, nil
	}

	// the context outlives the caller, hence it must not be request-scoped
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

	v := &Identity{
		log:     log,
		oc:      Oauth2Config(brand, country),
		ctx:     ctx,
		brand:   brand,
		user:    user,
		country: country,
		subject: subject,
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
		v.token = &token
		log.Redact(token.AccessToken, token.RefreshToken)
		if !settings.Exists(subject) {
			v.persist(&token)
		}
	}

	onlineC, err := providerauth.Register(subject, v)
	if err != nil {
		return nil, err
	}
	v.onlineC = onlineC
	v.setOnline(v.token.Valid())

	// add instance
	addInstance(subject, v)

	return v, nil
}

// ClientID returns the brand's oauth client id
func (v *Identity) ClientID() string {
	return v.oc.ClientID
}

// Token implements oauth2.TokenSource
func (v *Identity) Token() (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.token == nil {
		return nil, api.LoginRequiredError(v.subject)
	}

	if v.token.Valid() {
		return v.token, nil
	}

	token, err := v.oc.TokenSource(v.ctx, v.token).Token()
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			v.reset()
			return nil, api.LoginRequiredError(v.subject)
		}
		return nil, err
	}

	v.update(token)

	return token, nil
}

// StartChallenge implements api.AuthChallenger.
func (v *Identity) StartChallenge() (*api.AuthChallenge, error) {
	v.loginMu.Lock()
	defer v.loginMu.Unlock()

	if v.pending == nil {
		v.pending = &pendingLogin{
			oc:       Oauth2Config(v.brand, v.country),
			verifier: oauth2.GenerateVerifier(),
		}
	}

	return &api.AuthChallenge{
		Kind: api.AuthChallengeCode,
		Link: v.pending.oc.AuthCodeURL("", oauth2.S256ChallengeOption(v.pending.verifier)),
	}, nil
}

// SubmitChallenge implements api.AuthChallenger.
func (v *Identity) SubmitChallenge(answer string) (*api.AuthChallenge, error) {
	v.loginMu.Lock()
	defer v.loginMu.Unlock()

	if v.pending == nil {
		return nil, errors.New("no pending login")
	}

	code := authCode(answer)
	if code == "" {
		return nil, errors.New("missing authorization code")
	}

	token, err := v.pending.oc.Exchange(v.ctx, code, oauth2.VerifierOption(v.pending.verifier))
	if err != nil {
		return nil, err
	}

	v.pending = nil

	v.mu.Lock()
	defer v.mu.Unlock()
	v.update(token)

	return nil, nil
}

// Login implements api.AuthProvider. PSA uses the interactive login instead.
func (v *Identity) Login(string) (string, *oauth2.DeviceAuthResponse, error) {
	return "", nil, api.ErrNotAvailable
}

// HandleCallback implements api.AuthProvider. PSA uses the interactive login instead.
func (v *Identity) HandleCallback(url.Values) error {
	return api.ErrNotAvailable
}

// Logout implements api.AuthProvider
func (v *Identity) Logout() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.reset()

	return nil
}

// Authenticated implements api.AuthProvider
func (v *Identity) Authenticated() bool {
	token, err := v.Token()
	return err == nil && token.Valid()
}

// DisplayName implements api.AuthProvider
func (v *Identity) DisplayName() string {
	name, ok := brandNames[v.brand]
	if !ok {
		name = v.brand
	}
	return fmt.Sprintf("%s (%s)", name, v.user)
}

// reset drops the token. Caller must hold v.mu.
func (v *Identity) reset() {
	v.token = nil

	if settings.Exists(v.subject) {
		if err := settings.Delete(v.subject); err != nil {
			v.log.ERROR.Println(err)
		}
	}

	v.setOnline(false)
}

// update stores the token and signals online status. Caller must hold v.mu.
func (v *Identity) update(token *oauth2.Token) {
	v.token = token
	v.persist(token)
	v.setOnline(token.Valid())
}

func (v *Identity) persist(token *oauth2.Token) {
	v.log.Redact(token.AccessToken, token.RefreshToken)

	if err := settings.SetJson(v.subject, token); err != nil {
		v.log.ERROR.Printf("saving token: %v", err)
		return
	}

	// flush immediately so the token survives a restart right after login
	if err := settings.Persist(); err != nil {
		v.log.ERROR.Printf("persisting token: %v", err)
	}
}

// setOnline signals the auth handler without blocking (a blocking send under
// v.mu would deadlock via Authenticated()->Token()).
func (v *Identity) setOnline(online bool) {
	select {
	case v.onlineC <- online:
	default:
	}
}

// authCode accepts either the bare authorization code or the full redirect url
// the user was sent to
func authCode(s string) string {
	s = strings.TrimSpace(s)

	if u, err := url.Parse(s); err == nil && u.Query().Has("code") {
		return u.Query().Get("code")
	}

	return s
}
