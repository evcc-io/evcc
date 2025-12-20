package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("oauth", NewOAuthFromConfig)
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

type OAuth struct {
	mu      sync.Mutex
	log     *util.Logger
	oc      *oauth2.Config
	token   *oauth2.Token
	name    string
	devices []string
	subject string
	cv      string
	ctx     context.Context
	onlineC chan<- bool

	deviceFlow     bool
	tokenRetriever func(string, *oauth2.Token) error
	tokenStorer    func(*oauth2.Token) any
}

func NewOAuthFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		Name, Device  string
		oauth2.Config `mapstructure:",squash"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOAuth(ctx, cc.Name, cc.Device, &cc.Config)
}

var _ api.AuthProvider = (*OAuth)(nil)
var _ oauth2.TokenSource = (*OAuth)(nil)

func NewOAuth(ctx context.Context, name, device string, oc *oauth2.Config, opts ...func(o *OAuth)) (*OAuth, error) {
	if name == "" {
		return nil, errors.New("instance name must not be empty")
	}

	oauthMu.Lock()
	defer oauthMu.Unlock()

	// hash oauth2 config
	h := sha256.Sum256(fmt.Append(nil, oc))
	hash := hex.EncodeToString(h[:])[:8]
	subject := oc.ClientID + "-" + hash

	// reuse instance
	if instance := getInstance(subject); instance != nil {
		if device != "" && !slices.Contains(instance.devices, device) {
			instance.devices = append(instance.devices, device)
		}
		return instance, nil
	}

	log := util.NewLogger("oauth-" + hash)

	if client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); client == nil || !ok {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))
	}

	o := &OAuth{
		oc:      oc,
		log:     log,
		ctx:     ctx,
		subject: subject,
		name:    name,
	}

	for _, opt := range opts {
		opt(o)
	}

	if device != "" {
		o.devices = []string{device}
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

	if token.RefreshToken != "" {
		o.token = &token
	}

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

// Token
func (o *OAuth) Token() (*oauth2.Token, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.token == nil {
		return nil, api.LoginRequiredError(o.subject)
	}

	if o.token.Valid() {
		return o.token, nil
	}

	token, err := o.oc.TokenSource(o.ctx, o.token).Token()
	if err != nil {
		// force logout
		if strings.Contains(err.Error(), "invalid_") && settings.Exists(o.subject) {
			o.token = nil
			o.onlineC <- false
			settings.Delete(o.subject)
		}

		return nil, err
	}

	o.updateToken(token)

	return token, nil
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

	o.token = token

	o.onlineC <- token.Valid()
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
func (o *OAuth) Login(state string) (string, *oauth2.DeviceAuthResponse, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.cv = oauth2.GenerateVerifier()

	if o.deviceFlow {
		da, err := o.oc.DeviceAuth(o.ctx, oauth2.S256ChallengeOption(o.cv))
		if err != nil {
			return "", nil, err
		}

		go func() {
			ctx, cancel := context.WithTimeout(o.ctx, 5*time.Minute)
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

		return "", da, nil
	}

	if o.oc.Endpoint.AuthURL == "" {
		return "", nil, errors.New("missing auth url")
	}

	return o.oc.AuthCodeURL(state, oauth2.S256ChallengeOption(o.cv)), nil, nil
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

	o.token = nil
	o.onlineC <- false

	return nil
}

// DisplayName implements api.AuthProvider.
func (o *OAuth) DisplayName() string {
	if len(o.devices) > 0 {
		return fmt.Sprintf("%s (%s)", o.name, strings.Join(o.devices, ", "))
	}
	return o.name
}

// Authenticated implements api.AuthProvider.
func (o *OAuth) Authenticated() bool {
	token, err := o.Token()
	return err == nil && token.Valid()
}
