package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("demo", NewDemoFromConfig)
}

type demo struct {
	token       *oauth2.Token
	server      string
	method      string
	redirectUri string
	password    string
	onlineC     chan<- bool
}

var demoInstance *demo

// challengeDemo wraps demo as api.AuthChallenger for e2e tests. Method
// "challenge" mimics a scripted login (credentials, then a captcha), method
// "code" a vendor login page whose code the user pastes back.
type challengeDemo struct {
	*demo
	pending bool // captcha challenge outstanding
}

const demoCode = "demo-token" // issued by the simulator's mock login page

var _ api.AuthChallenger = (*challengeDemo)(nil)

const demoCaptcha = "1234"

// transparent svg showing "1234" with the usual captcha noise, like the real porsche captcha
const demoCaptchaImage = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyMDAiIGhlaWdodD0iNzAiIHZpZXdCb3g9IjAgMCAyMDAgNzAiPjxwYXRoIGQ9Ik0wIDUwUTUwIDEyIDEwMCA0MlQyMDAgMjIiIHN0cm9rZT0iI2I4YTU4YSIgc3Ryb2tlLXdpZHRoPSIyLjUiIGZpbGw9Im5vbmUiLz48cGF0aCBkPSJNMCAxOFE2MCA2NCAxMjAgMjZUMjAwIDU2IiBzdHJva2U9IiM4ZmEzYjgiIHN0cm9rZS13aWR0aD0iMiIgZmlsbD0ibm9uZSIvPjxnIGZvbnQtZmFtaWx5PSJHZW9yZ2lhLHNlcmlmIiBmb250LXNpemU9IjM4IiBmb250LXdlaWdodD0iYm9sZCI+PHRleHQgeD0iMjQiIHk9IjUwIiBmaWxsPSIjM2E1ZjhhIiB0cmFuc2Zvcm09InJvdGF0ZSgtMTIgMjQgNTApIj4xPC90ZXh0Pjx0ZXh0IHg9IjY2IiB5PSI0NSIgZmlsbD0iIzhhM2EzYSIgdHJhbnNmb3JtPSJyb3RhdGUoOCA2NiA0NSkiPjI8L3RleHQ+PHRleHQgeD0iMTA4IiB5PSI1MiIgZmlsbD0iIzNhN2E0YSIgdHJhbnNmb3JtPSJyb3RhdGUoLTYgMTA4IDUyKSI+MzwvdGV4dD48dGV4dCB4PSIxNDgiIHk9IjQ2IiBmaWxsPSIjNmE0YThhIiB0cmFuc2Zvcm09InJvdGF0ZSgxNCAxNDggNDYpIj40PC90ZXh0PjwvZz48ZyBmaWxsPSIjOTk5Ij48Y2lyY2xlIGN4PSIzOCIgY3k9IjE0IiByPSIxLjgiLz48Y2lyY2xlIGN4PSIxNzYiIGN5PSI1OCIgcj0iMS44Ii8+PGNpcmNsZSBjeD0iODgiIGN5PSI2MiIgcj0iMS44Ii8+PGNpcmNsZSBjeD0iMTMyIiBjeT0iMTAiIHI9IjEuOCIvPjxjaXJjbGUgY3g9IjEyIiBjeT0iNjAiIHI9IjEuOCIvPjwvZz48L3N2Zz4="

func NewDemoFromConfig(_ context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		Server      string
		Method      string
		RedirectUri string
		Secret      string
		Scope       string // advanced auth param, used by e2e tests
		User        string // challenge method only
		Password    string // challenge method only
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDemo(cc.Server, cc.Method, cc.RedirectUri, cc.Secret, cc.Password)
}

func NewDemo(server, method, redirectUri, secret, password string) (oauth2.TokenSource, error) {
	if secret != "topsecret" {
		return nil, errors.New("invalid secret")
	}

	// reuse instance (similar to oauth.go getInstance pattern)
	if demoInstance != nil {
		// update existing instance with new values
		demoInstance.server = server
		demoInstance.method = method
		demoInstance.redirectUri = redirectUri
		demoInstance.password = password
		return demoInstance, nil
	}

	demoInstance = &demo{
		server:      server,
		method:      method,
		redirectUri: redirectUri,
		password:    password,
	}

	var provider api.AuthProvider = demoInstance
	if method == "challenge" || method == "code" {
		provider = &challengeDemo{demo: demoInstance}
	}

	onlineC, err := providerauth.Register("demo", provider)
	if err != nil {
		return nil, err
	}
	demoInstance.onlineC = onlineC

	// Send initial auth status
	demoInstance.setOnline(false)

	return demoInstance, nil
}

// setOnline notifies the auth handler without blocking; see OAuth.setOnline.
func (o *demo) setOnline(online bool) {
	if o.onlineC == nil {
		return
	}
	select {
	case o.onlineC <- online:
	default:
	}
}

func (o *demo) Token() (*oauth2.Token, error) {
	if o.token == nil {
		return nil, api.LoginRequiredError("demo")
	}
	return o.token, nil
}

func (o *demo) Login(state string) (string, *oauth2.DeviceAuthResponse, error) {
	// Validate server URL has proper scheme
	if !strings.HasPrefix(o.server, "http://") && !strings.HasPrefix(o.server, "https://") {
		return "", nil, fmt.Errorf("server must start with http:// or https://")
	}

	// Validate redirect URI has proper scheme
	if !strings.HasPrefix(o.redirectUri, "http://") && !strings.HasPrefix(o.redirectUri, "https://") {
		return "", nil, fmt.Errorf("redirectUri must start with http:// or https://")
	}

	// Build mock login URL with state and redirectUri (complete callback URL)
	values := url.Values{}
	values.Set("state", state)
	values.Set("redirectUri", o.redirectUri)

	mockLoginURL := fmt.Sprintf("%s/mock-login?%s", o.server, values.Encode())

	if o.method == "device-code" {
		// Device code flow: URI comes from DeviceAuthResponse
		return "", &oauth2.DeviceAuthResponse{
			UserCode:        "12AB345",
			VerificationURI: mockLoginURL,
			Expiry:          time.Now().Add(10 * time.Minute),
		}, nil
	}

	// Redirect flow: URI in first return value
	return mockLoginURL, nil, nil
}

func (o *demo) Logout() error {
	o.token = nil
	o.setOnline(false)
	return nil
}

func (o *demo) HandleCallback(params url.Values) error {
	// Extract code from callback parameters
	code := params.Get("code")
	if code == "" {
		return fmt.Errorf("missing code parameter")
	}

	// Create token based on code (for demo, we use a fixed token)
	o.token = &oauth2.Token{
		AccessToken: code, // Use the code as the access token
		Expiry:      time.Now().Add(24 * time.Hour),
	}

	// Notify that authentication succeeded
	o.setOnline(true)

	return nil
}

func (o *demo) Authenticated() bool {
	return o.token != nil
}

func (o *demo) DisplayName() string {
	return "Demo Auth"
}

func demoCaptchaChallenge() *api.AuthChallenge {
	return &api.AuthChallenge{Kind: api.AuthChallengeCaptcha, Image: demoCaptchaImage}
}

// codeChallenge links to the mock login page, which redirects to a void page
// carrying the code, as vendors with a fixed redirect uri do
func (o *challengeDemo) codeChallenge() *api.AuthChallenge {
	values := url.Values{"state": {"evcc"}, "redirectUri": {o.server + "/void"}}
	return &api.AuthChallenge{
		Kind: api.AuthChallengeCode,
		Link: fmt.Sprintf("%s/mock-login?%s", o.server, values.Encode()),
	}
}

func (o *challengeDemo) StartChallenge() (*api.AuthChallenge, error) {
	if o.method == "code" {
		return o.codeChallenge(), nil
	}

	if o.password != "topsecret" {
		return nil, errors.New("invalid credentials")
	}
	o.pending = true
	return demoCaptchaChallenge(), nil
}

func (o *challengeDemo) SubmitChallenge(answer string) (*api.AuthChallenge, error) {
	if o.method == "code" {
		if codeFromAnswer(answer) != demoCode {
			return nil, errors.New("invalid code")
		}
		o.login(demoCode)
		return nil, nil
	}

	if !o.pending {
		return o.StartChallenge()
	}

	// wrong captcha: retry with a fresh challenge
	if answer != demoCaptcha {
		return demoCaptchaChallenge(), nil
	}

	o.pending = false
	o.login("challenge")
	return nil, nil
}

// codeFromAnswer accepts the bare code or the redirect url it was copied from
func codeFromAnswer(answer string) string {
	answer = strings.TrimSpace(answer)
	if u, err := url.Parse(answer); err == nil && u.Query().Has("code") {
		return u.Query().Get("code")
	}
	return answer
}

func (o *demo) login(token string) {
	o.token = &oauth2.Token{
		AccessToken: token,
		Expiry:      time.Now().Add(24 * time.Hour),
	}
	o.setOnline(true)
}
