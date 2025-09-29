package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"golang.org/x/oauth2"
)

type OAuth struct {
	oauth2.TokenSource
	mu      sync.Mutex
	cc      oauth2.Config
	subject string
	cv      string
	log     *util.Logger
	ctx     context.Context
}

var (
	// oauthMu    sync.Mutex
	identities = make(map[string]*OAuth)
)

func getInstance(subject string) *OAuth {
	return identities[subject]
}

func addInstance(subject string, identity *OAuth) {
	identities[subject] = identity
}

/* func init() {
	registry.AddCtx("oauth", NewOauthFromConfig)
}

func NewOauthFromConfig(ctx context.Context, other map[string]any) (Authorizer, error) {
	oauthMu.Lock()
	defer oauthMu.Unlock()
	// parse oauth config from yaml
	var cc oauth2.Config
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOauth(ctx, cc)
} */

func NewOauth(ctx context.Context, cc oauth2.Config, instanceName string) (*OAuth, error) {
	log := util.NewLogger("oauth-generic")

	if instanceName == "" {
		return nil, errors.New("instance name must not be empty")
	}

	// generate json string from oauth2 config
	bytejson, _ := json.Marshal(cc)

	h := sha256.New()
	h.Write(bytejson)
	fullHash := hex.EncodeToString(h.Sum(nil))
	sha256_hash := fullHash[:8]

	subject := instanceName + " (" + sha256_hash + ")"

	// reuse instance
	if instance := getInstance(subject); instance != nil {
		return instance, nil
	}

	// create new instance
	o := &OAuth{
		subject: subject,
		cc:      cc,
		log:     log,
		ctx:     ctx,
	}

	// load token from db
	var tok oauth2.Token
	if settings.Exists(o.subject) {
		o.log.DEBUG.Printf("loading token for %s from database", o.subject)

		if err := settings.Json(o.subject, &tok); err != nil {
			return nil, err
		}
	}

	o.TokenSource = oauth.RefreshTokenSource(&tok, o)

	// add instance
	addInstance(o.subject, o)

	// register authredirect
	providerauth.Register(o, subject)

	return o, nil
}

func (o *OAuth) Transport(base http.RoundTripper) http.RoundTripper {
	transport := oauth2.Transport{
		Base:   base,
		Source: o,
	}
	return &transport
}

// RefreshToken implements oauth.RefreshTokenSource.
func (o *OAuth) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	if token.RefreshToken == "" {
		return nil, api.ErrMissingToken
	}

	// log token before refresh
	o.log.DEBUG.Printf("refreshing token for %s", o.subject)

	// refresh token source
	token, err := o.cc.TokenSource(o.ctx, token).Token()
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			if settings.Exists(o.subject) {
				settings.Delete(o.subject)
			}
		}
		return nil, err
	}
	err = settings.SetJson(o.subject, token)

	return token, err
}

// HandleCallback implements api.AuthProvider.
func (o *OAuth) HandleCallback(responseValues url.Values) error {
	code := responseValues.Get("code")

	o.mu.Lock()
	defer o.mu.Unlock()

	token, err := o.cc.Exchange(o.ctx, code, oauth2.VerifierOption(o.cv))
	if err != nil {
		o.log.ERROR.Printf("error during oauth exchange: %s", err)
		return err
	}
	err = settings.SetJson(o.subject, token)
	if err != nil {
		o.log.ERROR.Printf("error saving token: %s", err)
	}

	o.TokenSource = oauth.RefreshTokenSource(token, o)
	return nil
}

// Login implements api.AuthProvider.
func (o *OAuth) Login(state string) string {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.cv = oauth2.GenerateVerifier()
	return o.cc.AuthCodeURL(state, oauth2.S256ChallengeOption(o.cv))
}

// Logout implements api.AuthProvider.
func (o *OAuth) Logout() error {
	o.log.INFO.Printf("removing %s from database", o.subject)
	if settings.Exists(o.subject) {
		settings.Delete(o.subject)
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.TokenSource = oauth.RefreshTokenSource(nil, o)
	return nil
}

// DisplayName implements api.AuthProvider.
func (o *OAuth) DisplayName() string {
	return o.subject
}

// Authenticated implements api.AuthProvider.
func (o *OAuth) Authenticated() bool {
	// check if token is valid
	if token, err := o.TokenSource.Token(); err == nil {
		return token.Valid()
	} else {
		return false
	}
}
