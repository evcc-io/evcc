package vehicle

import (
	"context"
	"fmt"
	"net/http"
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
	"github.com/google/uuid"
	mazda "github.com/tah78/mazda-go"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("mazda", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			User, Password string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		if cc.User == "" || cc.Password == "" {
			return nil, api.ErrMissingCredentials
		}

		return newMazdaIdentity(util.NewLogger("mazda"), cc.User, cc.Password)
	})
}

var (
	mazdaMu         sync.Mutex
	mazdaIdentities = make(map[string]*mazdaIdentity)
)

// mazdaIdentity is the Mazda account auth provider. The login runs server-side
// with the stored credentials, only the emailed verification code needs the user.
type mazdaIdentity struct {
	mu      sync.Mutex
	loginMu sync.Mutex

	log      *util.Logger
	redact   func(...string)
	client   *mazda.Client
	subject  string
	user     string
	password string
	tokenKey string

	token   *oauth2.Token
	onlineC chan<- bool
	pending *mazda.LoginSession
}

var (
	_ api.AuthProvider    = (*mazdaIdentity)(nil)
	_ api.AuthChallenger  = (*mazdaIdentity)(nil)
	_ oauth2.TokenSource  = (*mazdaIdentity)(nil)
	_ mazda.TokenProvider = (*mazdaIdentity)(nil)
)

// newMazdaIdentity returns the identity for the account, registering it with
// the provider-auth handler on first creation. Changed credentials are picked up.
func newMazdaIdentity(log *util.Logger, user, password string) (*mazdaIdentity, error) {
	mazdaMu.Lock()
	defer mazdaMu.Unlock()

	log.Redact(user, password)

	subject := "mazda." + strings.ToLower(user)

	if o, ok := mazdaIdentities[subject]; ok {
		o.log.Redact(user, password)

		o.mu.Lock()
		o.password = password
		o.mu.Unlock()
		return o, nil
	}

	deviceKey := subject + ".device"
	deviceID := uuid.NewString()

	if settings.Exists(deviceKey) {
		if err := settings.Json(deviceKey, &deviceID); err != nil {
			return nil, err
		}
	} else if err := persistJson(deviceKey, deviceID); err != nil {
		return nil, fmt.Errorf("saving device id: %w", err)
	}

	authHTTP := &http.Client{
		Timeout:   request.Timeout,
		Transport: transport.Default(),
	}

	o := &mazdaIdentity{
		log:      log,
		redact:   log.RotatingSlot(),
		subject:  subject,
		user:     user,
		password: password,
		tokenKey: subject + ".token",
	}

	o.client = mazda.NewClient(request.NewClient(log), authHTTP, deviceID, o)

	var token oauth2.Token
	if err := settings.Json(o.tokenKey, &token); err == nil && token.AccessToken != "" {
		o.token = &token
		o.redact(token.AccessToken, token.RefreshToken)
	}

	onlineC, err := providerauth.Register(subject, o)
	if err != nil {
		return nil, err
	}
	o.onlineC = onlineC

	o.mu.Lock()
	o.setOnlineLocked(o.token != nil)
	o.mu.Unlock()

	mazdaIdentities[subject] = o

	return o, nil
}

// Token implements oauth2.TokenSource. The token has no expiry, the api
// rejects it when it is no longer valid.
func (o *mazdaIdentity) Token() (*oauth2.Token, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.token == nil {
		return nil, api.LoginRequiredError(o.subject)
	}

	return o.token, nil
}

// AccessToken implements mazda.TokenProvider.
func (o *mazdaIdentity) AccessToken(context.Context) (string, error) {
	token, err := o.Token()
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

// Refresh implements mazda.TokenProvider.
func (o *mazdaIdentity) Refresh(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.token == nil {
		return api.LoginRequiredError(o.subject)
	}

	res, err := o.client.Refresh(ctx, o.token.AccessToken, o.token.RefreshToken)
	if err != nil {
		if mazda.IsAuthError(err) {
			o.log.DEBUG.Printf("token refresh rejected: %v", err)
			o.clearLocked()
			return api.LoginRequiredError(o.subject)
		}
		return err
	}

	return o.setTokenLocked(res)
}

// setTokenLocked persists and activates the token. Caller must hold o.mu.
func (o *mazdaIdentity) setTokenLocked(token *oauth2.Token) error {
	// a failed write keeps the previous token
	if err := persistJson(o.tokenKey, token); err != nil {
		return err
	}

	o.token = token
	o.redact(token.AccessToken, token.RefreshToken)
	o.setOnlineLocked(true)

	return nil
}

// clearLocked drops the token in memory and in settings. Caller must hold o.mu.
func (o *mazdaIdentity) clearLocked() {
	o.token = nil
	o.setOnlineLocked(false)

	if err := settings.Delete(o.tokenKey); err != nil {
		o.log.ERROR.Printf("deleting token: %v", err)
	}
}

// Login implements api.AuthProvider. Mazda uses the challenge login instead.
func (o *mazdaIdentity) Login(string) (string, *oauth2.DeviceAuthResponse, error) {
	return "", nil, api.ErrNotAvailable
}

// HandleCallback implements api.AuthProvider. Mazda uses the challenge login instead.
func (o *mazdaIdentity) HandleCallback(url.Values) error {
	return api.ErrNotAvailable
}

// Logout implements api.AuthProvider. The device id is kept so a later login
// does not need another verification code.
func (o *mazdaIdentity) Logout() error {
	o.loginMu.Lock()
	o.pending = nil
	o.loginMu.Unlock()

	o.mu.Lock()
	defer o.mu.Unlock()

	o.clearLocked()

	return nil
}

// Authenticated implements api.AuthProvider.
func (o *mazdaIdentity) Authenticated() bool {
	_, err := o.Token()
	return err == nil
}

// DisplayName implements api.AuthProvider.
func (o *mazdaIdentity) DisplayName() string {
	return "Mazda (" + o.user + ")"
}

// StartChallenge implements api.AuthChallenger. A pending login is re-served
// instead of restarted, the ui may call this several times.
func (o *mazdaIdentity) StartChallenge() (*api.AuthChallenge, error) {
	o.loginMu.Lock()
	defer o.loginMu.Unlock()

	return o.startLocked()
}

// SubmitChallenge implements api.AuthChallenger. A wrong code keeps the pending
// login so the user can retry.
func (o *mazdaIdentity) SubmitChallenge(answer string) (*api.AuthChallenge, error) {
	o.loginMu.Lock()
	defer o.loginMu.Unlock()

	if o.pending == nil || o.pending.Expired() {
		return o.startLocked()
	}

	token, err := o.client.Verify(context.Background(), o.pending, answer)
	if err != nil {
		return nil, err
	}

	if err := o.finishToken(token); err != nil {
		return nil, err
	}

	return nil, nil
}

// startLocked begins a new login. Caller must hold o.loginMu.
func (o *mazdaIdentity) startLocked() (*api.AuthChallenge, error) {
	if o.pending != nil && !o.pending.Expired() {
		return &api.AuthChallenge{Kind: api.AuthChallengeCode}, nil
	}

	o.pending = nil

	o.mu.Lock()
	user, password := o.user, o.password
	o.mu.Unlock()

	s, err := o.client.StartLogin(context.Background(), user, password)
	if err != nil {
		return nil, err
	}

	if s.Complete() {
		if err := o.finishToken(s.Token); err != nil {
			return nil, err
		}
		return nil, nil
	}

	o.pending = s

	return &api.AuthChallenge{Kind: api.AuthChallengeCode}, nil
}

// finishToken stores the token of a completed login. Caller must hold o.loginMu.
func (o *mazdaIdentity) finishToken(token *oauth2.Token) error {
	o.pending = nil

	if token == nil || token.AccessToken == "" || token.RefreshToken == "" {
		return api.LoginRequiredError(o.subject)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	return o.setTokenLocked(token)
}

func persistJson(key string, val any) error {
	if err := settings.SetJson(key, val); err != nil {
		return err
	}

	return settings.Persist()
}

// setOnlineLocked signals the auth handler. Caller must hold o.mu. The send is
// non-blocking, so a slow handler cannot deadlock via Authenticated()->Token().
func (o *mazdaIdentity) setOnlineLocked(online bool) {
	select {
	case o.onlineC <- online:
	default:
	}
}
