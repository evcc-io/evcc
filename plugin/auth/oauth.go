package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

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
	log     *util.Logger
	oc      *oauth2.Config
	subject string
	cv      string
	ctx     context.Context
	onlineC chan<- bool

	deviceFlow     bool
	tokenRetriever func(string, *oauth2.Token) error
	tokenStorer    func(*oauth2.Token) any
}

type oauthOption func(*OAuth)

func WithOauthDeviceFlowOption() func(o *OAuth) {
	return func(o *OAuth) {
		o.deviceFlow = true
	}
}

func WithTokenStorerOption(ts func(*oauth2.Token) any) func(o *OAuth) {
	return func(o *OAuth) {
		o.tokenStorer = ts
	}
}

func WithTokenRetrieverOption(tr func(string, *oauth2.Token) error) func(o *OAuth) {
	return func(o *OAuth) {
		o.tokenRetriever = tr
	}
}

var (
	oauthMu    sync.Mutex
	identities = make(map[string]*OAuth)
)

func getInstance(subject string) *OAuth {
	oauthMu.Lock()
	defer oauthMu.Unlock()
	return identities[subject]
}

func addInstance(subject string, identity *OAuth) {
	oauthMu.Lock()
	defer oauthMu.Unlock()
	identities[subject] = identity
}

func init() {
	registry.AddCtx("oauth", NewOauthFromConfig)
}

func NewOauthFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		Name          string
		oauth2.Config `mapstructure:",squash"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOauth(ctx, cc.Name, &cc.Config)
}

var (
	_ oauth.TokenRefresher = (*OAuth)(nil)
	_ api.AuthProvider     = (*OAuth)(nil)
)

func NewOauth(ctx context.Context, name string, oc *oauth2.Config, opts ...oauthOption) (oauth2.TokenSource, error) {
	if name == "" {
		return nil, errors.New("instance name must not be empty")
	}

	// hash oauth2 config
	h := sha256.Sum256(fmt.Append(nil, oc))
	hash := hex.EncodeToString(h[:])[:8]
	subject := name + " (" + hash + ")"

	// reuse instance
	if instance := getInstance(subject); instance != nil {
		return instance, nil
	}

	// create new instance
	o := &OAuth{
		subject: subject,
		oc:      oc,
		log:     util.NewLogger("oauth"),
		ctx:     ctx,
	}

	for _, opt := range opts {
		opt(o)
	}

	// load token from db
	var token oauth2.Token
	if settings.Exists(o.subject) {
		o.log.DEBUG.Printf("loading token for %s from database", o.subject)

		if o.tokenRetriever != nil {
			plain, err := settings.String(o.subject)
			if err != nil {
				return nil, err
			}

			if err := o.tokenRetriever(plain, &token); err != nil {
				return nil, err
			}
		} else {
			if err := settings.Json(o.subject, &token); err != nil {
				return nil, err
			}
		}
	}

	o.TokenSource = oauth.RefreshTokenSource(&token, o)

	// register auth redirect
	onlineC, err := providerauth.Register(subject, o)
	if err != nil {
		return nil, err
	}
	o.onlineC = onlineC

	o.onlineC <- token.Valid()

	// add instance
	addInstance(o.subject, o)

	return o, nil
}

// RefreshToken implements oauth.TokenRefresher.
func (o *OAuth) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	if token.RefreshToken == "" {
		return nil, api.ErrMissingToken
	}

	o.log.DEBUG.Printf("refreshing token for %s", o.subject)

	// refresh token source
	token, err := o.oc.TokenSource(o.ctx, token).Token()
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") && settings.Exists(o.subject) {
			settings.Delete(o.subject)
		}

		return nil, err
	}

	err = settings.SetJson(o.subject, token)

	return token, err
}

// updateToken must only be called when lock is held
func (o *OAuth) updateToken(token *oauth2.Token) {
	var store any = token

	// tokenStorer allows persisting the token together with it's extra properties
	if o.tokenStorer != nil {
		store = o.tokenStorer(token)
	}

	if err := settings.SetJson(o.subject, store); err != nil {
		o.log.ERROR.Printf("error saving token: %v", err)
	}

	o.onlineC <- token.Valid()

	o.TokenSource = oauth.RefreshTokenSource(token, o)
}

// HandleCallback implements api.AuthProvider.
func (o *OAuth) HandleCallback(params url.Values) error {
	code := params.Get("code")

	o.mu.Lock()
	defer o.mu.Unlock()

	token, err := o.oc.Exchange(o.ctx, code, oauth2.VerifierOption(o.cv))
	if err != nil {
		return err
	}

	o.updateToken(token)

	return nil
}

// Login implements api.AuthProvider.
func (o *OAuth) Login(state string) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.cv = oauth2.GenerateVerifier()

	if o.deviceFlow {
		ctx := context.Background()

		da, err := o.oc.DeviceAuth(ctx, oauth2.S256ChallengeOption(o.cv))
		if err != nil {
			return "", err
		}

		go func() {
			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			token, err := o.oc.DeviceAccessToken(ctx, da, oauth2.VerifierOption(o.cv))
			if err != nil {
				o.log.ERROR.Printf("error retrieving token: %v", err)
				return
			}

			o.mu.Lock()
			defer o.mu.Unlock()

			o.updateToken(token)
		}()

		return da.VerificationURIComplete, nil
	}

	if o.oc.Endpoint.AuthURL == "" {
		return "", errors.New("missing auth url")
	}

	return o.oc.AuthCodeURL(state, oauth2.S256ChallengeOption(o.cv)), nil
}

// Logout implements api.AuthProvider.
func (o *OAuth) Logout() error {
	o.log.DEBUG.Printf("removing %s from database", o.subject)

	if settings.Exists(o.subject) {
		if err := settings.Delete(o.subject); err != nil {
			o.log.ERROR.Println(err)
		}
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
	token, err := o.Token()
	return err == nil && token.Valid()
}
