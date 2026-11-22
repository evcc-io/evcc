package livewire

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

const (
	GigyaAPIKey = "4_6aX8cf8RFVt6F3JQQZaF3A"
	DataCenter  = "us1"
)

var (
	GigyaURL = "https://accounts.us1.gigya.com"

	// loginInterval throttles repeated logins after a failed login
	loginInterval = 5 * time.Minute
)

// clientHeaders mirror what the mobile app sends. The api calls work without
// them, only the session request still sends them as it was not tested otherwise.
var clientHeaders = map[string]string{
	"User-Agent":      "Android",
	"Content-Type":    request.JSONContent,
	"Accept-Language": "en-US",
	"model":           "Pixel 8",
	"osVersion":       "34",
}

// Identity performs the Gigya login and the LiveWire session exchange. There is
// no refresh flow and the JWT carries no expiry: it is used until the backend
// rejects it, which triggers a full re-login.
type Identity struct {
	*request.Helper
	log        *util.Logger
	redact     func(...string)
	user       string
	password   string
	deviceUUID string

	mu          sync.Mutex
	token       string
	lastAttempt time.Time
	lastErr     error
}

// NewIdentity creates a LiveWire identity for the given account and paired device uuid
func NewIdentity(log *util.Logger, user, password, deviceUUID string) *Identity {
	return &Identity{
		Helper:     request.NewHelper(log),
		log:        log,
		redact:     log.RotatingSlot(),
		user:       user,
		password:   password,
		deviceUUID: deviceUUID,
	}
}

// Login performs the full login flow and caches the resulting JWT
func (v *Identity) Login() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.login()
}

// Token returns the JWT, logging in if required
func (v *Identity) Token() (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.token != "" {
		return v.token, nil
	}

	if err := v.login(); err != nil {
		return "", err
	}

	return v.token, nil
}

// invalidate drops the cached JWT if it is still the one that was rejected
func (v *Identity) invalidate(token string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.token == token {
		v.token = ""
	}
}

// login must be called with the mutex held
func (v *Identity) login() error {
	if v.lastErr != nil && time.Since(v.lastAttempt) < loginInterval {
		return fmt.Errorf("login throttled: %w", v.lastErr)
	}

	v.lastAttempt = time.Now()
	v.token = ""

	uid, err := v.gigyaUID()
	if err != nil {
		v.lastErr = err
		return fmt.Errorf("gigya login: %w", err)
	}

	token, err := v.session(uid)
	if err != nil {
		v.lastErr = err
		return fmt.Errorf("session: %w", err)
	}

	v.token = token
	v.lastErr = nil
	v.redact(token)

	v.log.DEBUG.Println("logged in")

	return nil
}

// gigyaUID resolves the account UID. Gigya only accepts form encoding and reports errors in a 200 body.
func (v *Identity) gigyaUID() (string, error) {
	data := url.Values{
		"apiKey":   {GigyaAPIKey},
		"loginID":  {v.user},
		"password": {v.password},
		"include":  {"profile,data"},
		"format":   {"json"},
	}

	req, err := request.New(http.MethodPost, GigyaURL+"/accounts.login", strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return "", err
	}

	var res gigyaResponse
	if err := v.DoJSON(req, &res); err != nil {
		if res.ErrorMessage != "" {
			return "", fmt.Errorf("%s (%d)", res.ErrorMessage, res.ErrorCode)
		}
		return "", err
	}

	if res.ErrorCode != 0 {
		return "", fmt.Errorf("%s (%d)", res.ErrorMessage, res.ErrorCode)
	}

	if res.UID == "" {
		return "", errors.New("missing UID")
	}

	return res.UID, nil
}

// session exchanges the Gigya UID for a LiveWire JWT
func (v *Identity) session(uid string) (string, error) {
	data := SessionRequest{
		UID:        uid,
		DeviceUUID: v.deviceUUID,
		DataCenter: DataCenter,
	}

	req, err := request.New(http.MethodPost, BaseURL+"/session", request.MarshalJSON(data), clientHeaders)
	if err != nil {
		return "", err
	}

	var res SessionResponse
	err = v.DoJSON(req, &res)
	if envErr := res.Err(); envErr != nil {
		return "", envErr
	}
	if err != nil {
		return "", err
	}

	if res.JWT == "" {
		return "", errors.New("missing jwt")
	}

	return res.JWT, nil
}

// Transport decorates requests with the bearer token and the brand query param.
// A 401 triggers one re-login and retry.
func (v *Identity) Transport(base http.RoundTripper) http.RoundTripper {
	return &transport.Decorator{
		Decorator: v.decorate,
		Base:      &retryTransport{identity: v, base: base},
	}
}

func (v *Identity) decorate(req *http.Request) error {
	token, err := v.Token()
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	q := req.URL.Query()
	if q.Get("brand") == "" {
		q.Set("brand", Brand)
		req.URL.RawQuery = q.Encode()
	}

	return nil
}

// retryTransport re-authenticates once when the backend rejects the token.
// All requests through it are GETs, so a retry needs no body handling.
type retryTransport struct {
	identity *Identity
	base     http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	resp.Body.Close()

	t.identity.invalidate(strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer "))

	token, err := t.identity.Token()
	if err != nil {
		return nil, err
	}

	retry := req.Clone(req.Context())
	retry.Header.Set("Authorization", "Bearer "+token)

	return t.base.RoundTrip(retry)
}
