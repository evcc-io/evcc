package auth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	zonneplan "github.com/jap/go-zonneplan"
	"golang.org/x/oauth2"
)

// The Zonneplan auth provider logs the user in with a one-time password
// mailed to the account address (challenge flow). The rotating token is
// persisted under the provider id and shared by every device using the same
// account.

func init() {
	registry.AddCtx("zonneplan", NewZonneplanFromConfig)
}

var (
	_ api.AuthProvider   = (*zonneplanIdentity)(nil)
	_ api.AuthChallenger = (*zonneplanIdentity)(nil)
	_ oauth2.TokenSource = (*zonneplanIdentity)(nil)
)

type zonneplanIdentity struct {
	mu      sync.Mutex
	loginMu sync.Mutex

	log      *util.Logger
	redact   func(...string)
	subject  string
	tokenKey string
	client   *zonneplan.Client

	token   *oauth2.Token
	onlineC chan<- bool
	pending *zonneplan.OtpChallenge
}

var (
	zonneplanMu         sync.Mutex
	zonneplanIdentities = make(map[string]*zonneplanIdentity)
)

func NewZonneplanFromConfig(_ context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		Email string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Email == "" {
		return nil, api.ErrMissingCredentials
	}

	return newZonneplanIdentity(util.NewLogger("zonneplan"), cc.Email)
}

// newZonneplanIdentity returns the identity for the account, registering it
// with the provider-auth handler on first creation.
func newZonneplanIdentity(log *util.Logger, email string) (*zonneplanIdentity, error) {
	zonneplanMu.Lock()
	defer zonneplanMu.Unlock()

	log.Redact(email)

	subject := "zonneplan." + strings.ToLower(email)

	if o, ok := zonneplanIdentities[subject]; ok {
		return o, nil
	}

	client := zonneplan.NewClient(zonneplan.WithEmail(email))

	o := &zonneplanIdentity{
		log:      log,
		redact:   log.RotatingSlot(),
		subject:  subject,
		tokenKey: subject + ".token",
		client:   client,
	}

	var token oauth2.Token
	if err := settings.Json(o.tokenKey, &token); err == nil && token.AccessToken != "" {
		o.token = &token
		o.redact(token.AccessToken, token.RefreshToken)
		client.SetToken(&zonneplan.Token{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		})
	}

	onlineC, err := providerauth.Register(subject, o)
	if err != nil {
		return nil, err
	}
	o.onlineC = onlineC

	o.setOnline(o.token != nil)

	zonneplanIdentities[subject] = o

	return o, nil
}

// setOnline notifies the auth handler without blocking; see OAuth.setOnline.
func (o *zonneplanIdentity) setOnline(online bool) {
	select {
	case o.onlineC <- online:
	default:
	}
}

// Token implements oauth2.TokenSource. It transparently refreshes an expiring
// token; the refresh token rotates, so the new token is persisted immediately.
func (o *zonneplanIdentity) Token() (*oauth2.Token, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.token.Valid() {
		return o.token, nil
	}

	if o.token == nil || o.token.RefreshToken == "" {
		return nil, api.LoginRequiredError(o.subject)
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	if err := o.client.RefreshToken(ctx); err != nil {
		if errors.Is(err, zonneplan.ErrAuth) {
			// the refresh token was rejected: the user has to log in again
			o.clearTokenLocked()
			return nil, api.LoginRequiredError(o.subject)
		}
		return nil, err
	}

	if err := o.storeTokenLocked(o.client.Token()); err != nil {
		return nil, err
	}

	return o.token, nil
}

// storeTokenLocked persists and activates the token. Caller must hold o.mu.
// Settings are flushed right away: a lost rotated token locks the user out.
func (o *zonneplanIdentity) storeTokenLocked(t *zonneplan.Token) error {
	token := &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry,
	}

	if err := settings.SetJson(o.tokenKey, token); err != nil {
		return err
	}
	if err := settings.Persist(); err != nil {
		return err
	}

	o.token = token
	o.redact(token.AccessToken, token.RefreshToken)
	o.setOnline(true)

	return nil
}

// clearTokenLocked drops the token in memory and in settings. Caller must hold o.mu.
func (o *zonneplanIdentity) clearTokenLocked() {
	o.token = nil
	o.setOnline(false)

	if err := settings.Delete(o.tokenKey); err != nil {
		o.log.ERROR.Printf("deleting token: %v", err)
	}
}

// Login implements api.AuthProvider. Zonneplan uses the challenge login instead.
func (o *zonneplanIdentity) Login(string) (string, *oauth2.DeviceAuthResponse, error) {
	return "", nil, api.ErrNotAvailable
}

// HandleCallback implements api.AuthProvider. Zonneplan uses the challenge login instead.
func (o *zonneplanIdentity) HandleCallback(url.Values) error {
	return api.ErrNotAvailable
}

// Logout implements api.AuthProvider. The pending challenge is dropped with
// the session.
func (o *zonneplanIdentity) Logout() error {
	o.loginMu.Lock()
	o.pending = nil
	o.loginMu.Unlock()

	o.mu.Lock()
	defer o.mu.Unlock()

	o.clearTokenLocked()

	return nil
}

// Authenticated implements api.AuthProvider.
func (o *zonneplanIdentity) Authenticated() bool {
	_, err := o.Token()
	return err == nil
}

// DisplayName implements api.AuthProvider.
func (o *zonneplanIdentity) DisplayName() string {
	return "Zonneplan"
}

// StartChallenge implements api.AuthChallenger. It has Zonneplan mail a
// one-time password to the account address, a new one on every call.
func (o *zonneplanIdentity) StartChallenge() (*api.AuthChallenge, error) {
	o.loginMu.Lock()
	defer o.loginMu.Unlock()

	return o.startLocked()
}

func (o *zonneplanIdentity) startLocked() (*api.AuthChallenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	challenge, err := o.client.RequestOTP(ctx, "evcc")
	if err != nil {
		return nil, err
	}
	o.pending = &challenge

	return &api.AuthChallenge{Kind: api.AuthChallengeCode}, nil
}

// SubmitChallenge implements api.AuthChallenger. A wrong code keeps the
// pending login so the user can retry.
func (o *zonneplanIdentity) SubmitChallenge(answer string) (*api.AuthChallenge, error) {
	o.loginMu.Lock()
	defer o.loginMu.Unlock()

	if o.pending == nil {
		return o.startLocked()
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	if err := o.client.SubmitOTP(ctx, *o.pending, answer); err != nil {
		return nil, err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if err := o.storeTokenLocked(o.client.Token()); err != nil {
		return nil, err
	}

	o.pending = nil

	return nil, nil
}
