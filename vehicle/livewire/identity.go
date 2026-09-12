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
	"github.com/golang-jwt/jwt/v5"
)

const (
	GigyaAPIKey = "4_6aX8cf8RFVt6F3JQQZaF3A"
	DataCenter  = "us1"
)

var (
	GigyaURL = "https://accounts.us1.gigya.com"

	// loginInterval throttles repeated logins after a failed login or after the
	// backend keeps rejecting freshly issued tokens
	loginInterval = 5 * time.Minute
)

// maxFastRejects is the number of tokens rejected within loginInterval of their
// issue before re-logins are throttled. One immediate re-login is always allowed.
const maxFastRejects = 1

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
// no refresh flow: an expired or rejected JWT triggers a full re-login.
type Identity struct {
	*request.Helper
	log        *util.Logger
	redact     func(...string)
	user       string
	password   string
	deviceUUID string

	mu          sync.Mutex
	token       string
	expiry      time.Time
	issued      time.Time
	lastAttempt time.Time
	lastErr     error
	fastRejects int
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

// Token returns a valid JWT, logging in if required
func (v *Identity) Token() (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.token != "" && (v.expiry.IsZero() || time.Now().Before(v.expiry)) {
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

	if v.token != token {
		return
	}

	v.token = ""

	// a token that lasted longer than the interval is a normal expiry, a
	// quick rejection counts towards throttling
	if time.Since(v.issued) < loginInterval {
		v.fastRejects++
	} else {
		v.fastRejects = 0
	}
}

// login must be called with the mutex held
func (v *Identity) login() error {
	if time.Since(v.lastAttempt) < loginInterval {
		if v.lastErr != nil {
			return fmt.Errorf("login throttled: %w", v.lastErr)
		}
		if v.fastRejects > maxFastRejects {
			return errors.New("login throttled: backend keeps rejecting fresh tokens")
		}
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
	v.expiry = tokenExpiry(token)
	v.issued = time.Now()
	v.lastErr = nil
	v.redact(token)

	if v.expiry.IsZero() {
		v.log.DEBUG.Println("logged in, token has no expiry")
	} else {
		v.log.DEBUG.Printf("logged in, token expires %v", v.expiry.Round(time.Second))
	}

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

// tokenExpiry derives the expiry from the JWT exp claim, with a safety margin.
// The live backend issues tokens without exp, then the zero time means unknown
// and the token is used until the backend rejects it.
func tokenExpiry(token string) time.Time {
	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser().ParseUnverified(token, &claims); err == nil && claims.ExpiresAt != nil {
		return claims.ExpiresAt.Add(-time.Minute)
	}
	return time.Time{}
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

// retryTransport re-authenticates once when the backend rejects the token
type retryTransport struct {
	identity *Identity
	base     http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || !replayable(req) {
		return resp, err
	}

	rejected := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	t.identity.invalidate(rejected)

	token, err := t.identity.Token()
	if err != nil {
		return resp, nil
	}

	resp.Body.Close()

	retry := req.Clone(req.Context())
	if req.GetBody != nil {
		if retry.Body, err = req.GetBody(); err != nil {
			return nil, err
		}
	}
	retry.Header.Set("Authorization", "Bearer "+token)

	return t.base.RoundTrip(retry)
}

// replayable reports whether the request body can be sent again
func replayable(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}
