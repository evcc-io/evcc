package psa

import (
	"context"
	"errors"
	"net/url"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Identity is a PSA brand auth provider. It implements api.InteractiveAuthProvider
// (browser login via evcc's providerauth) and oauth2.TokenSource (used by the API
// client), persisting the token in the settings database. A single Stellantis
// account serves all of a brand's vehicles, so the identity is a per-brand singleton.
type Identity struct {
	mu          sync.Mutex
	log         *util.Logger
	oc          *oauth2.Config
	ctx         context.Context
	token       *oauth2.Token
	onlineC     chan<- bool
	subject     string
	realm       string
	displayName string
}

var (
	_ api.AuthProvider            = (*Identity)(nil)
	_ api.InteractiveAuthProvider = (*Identity)(nil)
	_ oauth2.TokenSource          = (*Identity)(nil)
)

// NewIdentity returns the (per-brand singleton) PSA identity, registering it with
// evcc's provider-auth handler on first creation. A non-nil seed token (e.g. from
// a legacy `evcc token` config) is used when the database has none yet.
func NewIdentity(ctx context.Context, log *util.Logger, brand, realm, displayName string, oc *oauth2.Config, seed *oauth2.Token) (*Identity, error) {
	mu.Lock()
	defer mu.Unlock()

	subject := "psa." + brand
	if instance := getInstance(subject); instance != nil {
		return instance.(*Identity), nil
	}

	authCtx := context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	o := &Identity{
		log:         log,
		oc:          oc,
		ctx:         authCtx,
		subject:     subject,
		realm:       realm,
		displayName: displayName,
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

	addInstance(subject, o)
	return o, nil
}

// Token implements oauth2.TokenSource.
func (o *Identity) Token() (*oauth2.Token, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.token == nil {
		return nil, api.LoginRequiredError(o.subject)
	}
	if o.token.Valid() {
		return o.token, nil
	}
	if o.token.RefreshToken == "" {
		return nil, api.LoginRequiredError(o.subject)
	}

	token, err := o.oc.TokenSource(o.ctx, &oauth2.Token{RefreshToken: o.token.RefreshToken}).Token()
	if err != nil {
		return nil, err
	}

	o.update(token)
	return token, nil
}

// Login implements api.AuthProvider. PSA only supports interactive (credential)
// login, so the redirect flow is not used.
func (o *Identity) Login(string) (string, *oauth2.DeviceAuthResponse, error) {
	return "", nil, errors.New("interactive login only")
}

// HandleCallback implements api.AuthProvider (unused, see Login).
func (o *Identity) HandleCallback(url.Values) error {
	return errors.New("interactive login only")
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
	return o.displayName
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
