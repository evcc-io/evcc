package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db/settings"
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
	redact  func(...string)

	deviceFlow       bool
	tokenRetriever   func(string, *oauth2.Token) error
	tokenStorer      func(*oauth2.Token) any
	refreshErrLogged bool
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

	log := util.ContextLoggerWithDefault(ctx, util.NewLogger("oauth-"+hash))

	if client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); client == nil || !ok {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))
	}

	o := &OAuth{
		oc:      oc,
		log:     log,
		ctx:     ctx,
		subject: subject,
		name:    name,
		redact:  log.RotatingSlot(),
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
		o.redact(token.AccessToken, token.RefreshToken)
	}

	// register auth redirect
	onlineC, err := providerauth.Register(subject, o)
	if err != nil {
		return nil, err
	}
	o.onlineC = onlineC

	o.setOnline(token.Valid())

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
		if !o.refreshErrLogged {
			o.log.ERROR.Printf("token refresh failed: %v", err)
			o.refreshErrLogged = true
		}

		// force logout
		if strings.Contains(err.Error(), "invalid_") && settings.Exists(o.subject) {
			o.token = nil
			o.setOnline(false)
			settings.Delete(o.subject)
		}

		return nil, err
	}

	o.updateToken(token)

	return token, nil
}

const shortRefreshLifetime = 24 * time.Hour

func durationFromSeconds(sec int64) (time.Duration, bool) {
	if sec > math.MaxInt64/int64(time.Second) || sec < math.MinInt64/int64(time.Second) {
		return 0, false
	}
	return time.Duration(sec) * time.Second, true
}

func durationFromSecondsFloat(sec float64) (time.Duration, bool) {
	if math.IsNaN(sec) || math.IsInf(sec, 0) {
		return 0, false
	}
	maxSec := float64(math.MaxInt64 / int64(time.Second))
	minSec := float64(math.MinInt64 / int64(time.Second))
	if sec > maxSec || sec < minSec {
		return 0, false
	}
	return time.Duration(sec) * time.Second, true
}

func refreshExpiresIn(token *oauth2.Token) (time.Duration, bool) {
	if token == nil {
		return 0, false
	}

	switch v := token.Extra("refresh_expires_in").(type) {
	case nil:
		return 0, false
	case float64:
		return durationFromSecondsFloat(v)
	case int:
		return durationFromSeconds(int64(v))
	case int64:
		return durationFromSeconds(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return durationFromSeconds(n)
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return durationFromSeconds(n)
	default:
		return 0, false
	}
}

func (o *OAuth) warnShortRefresh(token *oauth2.Token) {
	d, ok := refreshExpiresIn(token)
	if !ok || d <= 0 || d >= shortRefreshLifetime {
		return
	}

	o.log.WARN.Printf("refresh token expires in %v; log in again if the session was not set to stay signed in", d.Round(time.Second))
}

// updateToken must only be called when lock is held
func (o *OAuth) updateToken(token *oauth2.Token) {
	o.refreshErrLogged = false
	o.warnShortRefresh(token)

	var store any = token

	// tokenStorer allows persisting the token together with it's extra properties
	if o.tokenStorer != nil {
		store = o.tokenStorer(token)
	}

	if err := settings.SetJson(o.subject, store); err != nil {
		o.log.ERROR.Printf("error saving token: %v", err)
	}

	o.token = token

	// keep the token out of the logs, e.g. the Authorization header when logging headers
	o.redact(token.AccessToken, token.RefreshToken)

	o.setOnline(token.Valid())
}

// setOnline signals the auth handler without blocking; the value is only a
// wakeup. A blocking send under o.mu would deadlock via Authenticated()->Token().
func (o *OAuth) setOnline(online bool) {
	select {
	case o.onlineC <- online:
	default:
	}
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
	o.setOnline(false)

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
