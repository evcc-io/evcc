package auth

// TODO
// - configurable redirect uri

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/oauth2redirect"
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
	oauthMu    sync.Mutex
	identities = make(map[string]*OAuth)
)

func getInstance(subject string) *OAuth {
	return identities[subject]
}

func addInstance(subject string, identity *OAuth) {
	identities[subject] = identity
}

func init() {
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
}

func NewOauth(ctx context.Context, cc oauth2.Config) (*OAuth, error) {
	// TODO subject should include hash of complete oauth2 config
	subject := "oauth." + cc.ClientID

	// reuse instance
	if instance := getInstance(subject); instance != nil {
		return instance, nil
	}

	log := util.NewLogger("oauth-generic")

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
		if err := settings.Json(o.subject, &tok); err != nil {
			return nil, err
		}
	}

	o.TokenSource = oauth.RefreshTokenSource(&tok, o)

	// add instance
	addInstance(o.subject, o)

	// register authredirect
	oauth2redirect.Register(o, subject)

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
	o.mu.Lock()
	defer o.mu.Unlock()

	if token.RefreshToken == "" {
		return nil, api.ErrMissingToken
	}

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

// AuthCodeURL implements api.AuthProvider.
func (o *OAuth) AuthCodeURL(state string) string {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.cv = oauth2.GenerateVerifier()
	return o.cc.AuthCodeURL(state, oauth2.S256ChallengeOption(o.cv))
}

// HandleCallback implements api.AuthProvider.
func (o *OAuth) HandleCallback(r *http.Request) {
	q := r.URL.Query()
	code := q.Get("code")

	o.mu.Lock()
	defer o.mu.Unlock()

	token, err := o.cc.Exchange(o.ctx, code, oauth2.VerifierOption(o.cv))
	if err != nil {
		o.log.ERROR.Printf("error during oauth exchange: %s", err)
		return
	}
	err = settings.SetJson(o.subject, token)
	if err != nil {
		o.log.ERROR.Printf("error saving token: %s", err)
	}

	o.TokenSource = oauth.RefreshTokenSource(token, o)
}

// HandleLogout implements api.AuthProvider.
func (o *OAuth) HandleLogout(r *http.Request) {
	o.log.INFO.Printf("removing %s from database", o.subject)
	if settings.Exists(o.subject) {
		settings.Delete(o.subject)
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.TokenSource = oauth.RefreshTokenSource(nil, o)
}
