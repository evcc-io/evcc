package porsche

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

var _ api.InteractiveAuthProvider = (*Identity)(nil)

// Server-side port of the Auth0 "Identifier First" login used by the My Porsche
// app, adapted from https://github.com/CJNE/pyporscheconnectapi. Because evcc
// performs the HTTP requests itself (rather than a browser), it can read the
// authorization code straight from the my-porsche-app:// redirect Location
// header. The only step that needs the user is solving the captcha.

var (
	// JWT/JSON in an atob("...") blob carrying the Auth0 ACUL screen context
	reAtob = regexp.MustCompile(`atob\("([A-Za-z0-9+/=]+)"`)
	// inline svg data URI fallback
	reSvg = regexp.MustCompile(`(data:image/svg\+xml[^"')\s]+)`)
)

// ErrCaptchaRequired is returned by the login steps when a captcha must be
// solved before the flow can continue. The image is a data URI (SVG).
type ErrCaptchaRequired struct {
	Image string
}

func (e *ErrCaptchaRequired) Error() string { return "captcha required" }

// LoginSession holds the state of an in-progress interactive login. It keeps
// the HTTP client (and its cookie jar) and the Auth0 state across the captcha
// challenge so the flow can be resumed.
type LoginSession struct {
	log      *util.Logger
	client   *http.Client
	state    string
	email    string
	password string
	created  time.Time
}

// NewLoginSession starts an interactive login. It returns the authorization
// code on success, or an *ErrCaptchaRequired if a captcha must be solved (call
// SolveCaptcha to continue).
func NewLoginSession(log *util.Logger, email, password string) (*LoginSession, string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, "", err
	}

	client := request.NewClient(log)
	client.Jar = jar
	// do not follow redirects: we need to read Location headers
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	s := &LoginSession{
		log:      log,
		client:   client,
		email:    email,
		password: password,
		created:  time.Now(),
	}

	code, err := s.start()
	return s, code, err
}

func headers() map[string]string {
	return map[string]string{
		"User-Agent":  "evcc",
		"X-Client-ID": XClientID,
	}
}

// start performs the initial /authorize request and the identifier step.
func (s *LoginSession) start() (string, error) {
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {ClientID},
		"redirect_uri":  {RedirectURI},
		"audience":      {Audience},
		"scope":         {strings.Join(Scopes, " ")},
		"state":         {"evcc"},
	}

	loc, err := s.getLocation(OAuthURI+"/authorize?"+v.Encode(), headers())
	if err != nil {
		return "", err
	}

	// existing Auth0 session -> code returned directly
	if code := loc.Query().Get("code"); code != "" {
		return code, nil
	}

	state := loc.Query().Get("state")
	if state == "" {
		return "", errors.New("login: no state in authorize redirect")
	}
	s.state = state

	return s.identifier("")
}

// identifier submits the e-mail (and optionally a captcha solution).
func (s *LoginSession) identifier(captcha string) (string, error) {
	data := url.Values{
		"state":                       {s.state},
		"username":                    {s.email},
		"js-available":                {"true"},
		"webauthn-available":          {"false"},
		"is-brave":                    {"false"},
		"webauthn-platform-available": {"false"},
		"action":                      {"default"},
	}
	if captcha != "" {
		data.Set("captcha", captcha)
	}

	uri := OAuthURI + "/u/login/identifier?" + url.Values{"state": {s.state}}.Encode()
	resp, err := s.post(uri, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "", errors.New("login: wrong credentials")
	case http.StatusBadRequest:
		body, _ := io.ReadAll(resp.Body)
		img := extractCaptcha(string(body))
		if img == "" {
			return "", errors.New("login: captcha required but image not found")
		}
		return "", &ErrCaptchaRequired{Image: img}
	}

	// success -> password step
	return s.submitPassword()
}

// password submits the password and resumes the authorize flow to obtain the code.
func (s *LoginSession) submitPassword() (string, error) {
	data := url.Values{
		"state":    {s.state},
		"username": {s.email},
		"password": {s.password},
		"action":   {"default"},
	}

	uri := OAuthURI + "/u/login/password?" + url.Values{"state": {s.state}}.Encode()
	resp, err := s.post(uri, data)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		return "", errors.New("login: wrong password")
	}

	resume := resp.Header.Get("Location")
	if resume == "" {
		return "", fmt.Errorf("login: no resume location (status %d)", resp.StatusCode)
	}
	if strings.HasPrefix(resume, "/") {
		resume = OAuthURI + resume
	}

	loc, err := s.getLocation(resume, headers())
	if err != nil {
		return "", err
	}

	code := loc.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("login: no code in callback (%s)", loc.Redacted())
	}
	return code, nil
}

// SolveCaptcha continues a paused login with the user-provided captcha solution.
// Returns the authorization code, or another *ErrCaptchaRequired if it was wrong.
func (s *LoginSession) SolveCaptcha(solution string) (string, error) {
	return s.identifier(solution)
}

// getLocation issues a GET and returns the parsed Location header (expects 3xx).
func (s *LoginSession) getLocation(uri string, hdr map[string]string) (*url.URL, error) {
	req, err := request.New(http.MethodGet, uri, nil, hdr)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return nil, fmt.Errorf("login: expected redirect, got status %d", resp.StatusCode)
	}
	return url.Parse(loc)
}

// post submits a form with the standard headers.
func (s *LoginSession) post(uri string, data url.Values) (*http.Response, error) {
	h := headers()
	h["Content-Type"] = request.FormContent
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), h)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req)
}

// extractCaptcha pulls the captcha image (data URI) from the Auth0 login HTML.
func extractCaptcha(html string) string {
	if m := reAtob.FindStringSubmatch(html); m != nil {
		if decoded, err := base64.StdEncoding.DecodeString(m[1]); err == nil {
			var ctx struct {
				Screen struct {
					Captcha struct {
						Image string `json:"image"`
					} `json:"captcha"`
				} `json:"screen"`
			}
			if json.Unmarshal(decoded, &ctx) == nil && ctx.Screen.Captcha.Image != "" {
				return ctx.Screen.Captcha.Image
			}
		}
	}
	if m := reSvg.FindStringSubmatch(html); m != nil {
		return m[1]
	}
	return ""
}

// ExchangeCode exchanges an authorization code for a token using the identity's
// OAuth config (and X-Client-ID via its context).
func (o *Identity) ExchangeCode(code string) (*oauth2.Token, error) {
	token, err := o.oc.Exchange(o.ctx, code)
	if err != nil {
		return nil, err
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.update(token)
	return token, nil
}

// Expired reports whether the interactive login session is too old to resume.
func (s *LoginSession) Expired() bool {
	return time.Since(s.created) > 5*time.Minute
}

// Challenge implements api.InteractiveAuthProvider: the initial login form.
func (o *Identity) Challenge() *api.AuthChallenge {
	return &api.AuthChallenge{
		Fields: []api.AuthField{
			{Name: "email", Type: "email"},
			{Name: "password", Type: "password"},
		},
	}
}

// Submit implements api.InteractiveAuthProvider. It either starts a login with
// the submitted credentials or answers a pending captcha, returning the next
// challenge (a captcha) or nil once authentication succeeded.
func (o *Identity) Submit(values map[string]string) (*api.AuthChallenge, error) {
	o.loginMu.Lock()
	defer o.loginMu.Unlock()

	var code string
	var err error

	if o.pending != nil && !o.pending.Expired() {
		// answer the pending captcha
		code, err = o.pending.SolveCaptcha(values["captcha"])
	} else {
		// start a new login with the submitted credentials
		o.pending, code, err = NewLoginSession(o.log, values["email"], values["password"])
	}

	var capErr *ErrCaptchaRequired
	if errors.As(err, &capErr) {
		return &api.AuthChallenge{
			Image:  capErr.Image,
			Fields: []api.AuthField{{Name: "captcha", Type: "text"}},
		}, nil
	}
	if err != nil {
		o.pending = nil
		return nil, err
	}

	// success - exchange the authorization code for a token
	if _, err := o.ExchangeCode(code); err != nil {
		o.pending = nil
		return nil, err
	}
	o.pending = nil
	return nil, nil
}
