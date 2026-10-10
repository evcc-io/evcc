package eudataact

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUserPage(t *testing.T) {
	for _, tc := range []struct {
		name string
		uri  string
		want bool
	}{
		{"German", BaseURL + "/content/euda/de/de/user.html", true},
		{"English", BaseURL + "/content/euda/gb/en/user.html", true},
		{"query", BaseURL + "/content/euda/de/en/user.html?next=/error", true},
		{"fragment", BaseURL + "/content/euda/fr/fr/user.html#profile", true},
		{"error page", BaseURL + "/content/euda/de/de/error.html?next=user.html", false},
		{"other content", BaseURL + "/content/other/de/de/user.html", false},
		{"lookalike prefix", BaseURL + "/content/euda-other/de/de/user.html", false},
		{"lookalike filename", BaseURL + "/content/euda/de/de/notuser.html", false},
		{"extra segment", BaseURL + "/content/euda/de/de/user.html/extra", false},
		{"trailing slash", BaseURL + "/content/euda/de/de/user.html/", false},
		{"wrong host", "https://example.org/content/euda/de/de/user.html", false},
		{"lookalike host", BaseURL + ".example.org/content/euda/de/de/user.html", false},
		{"HTTP", "http://" + portalHost + "/content/euda/de/de/user.html", false},
		{"relative", "/content/euda/de/de/user.html", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.uri)
			require.NoError(t, err)
			assert.Equal(t, tc.want, isUserPage(u))
		})
	}
}

type loginRoundTripFunc func(*http.Request) (*http.Response, error)

func (f loginRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type loginTestBody struct {
	io.Reader
	closed bool
}

func (b *loginTestBody) Close() error {
	b.closed = true
	return nil
}

func TestLoginRedirects(t *testing.T) {
	const (
		userPage         = BaseURL + "/content/euda/de/de/user.html"
		portalCallback   = BaseURL + "/services/callbacklogin"
		marketingPage    = vwidentity.BaseURL + "/consent/marketing/test"
		identityCallback = vwidentity.BaseURL + "/oidc/v1/oauth/client/callback"
		identifier       = vwidentity.BaseURL + "/login/identifier"
		authenticate     = vwidentity.BaseURL + "/login/authenticate"
		signin           = vwidentity.BaseURL + "/signin-service/v1/signin/test"
		vehicles         = BaseURL + "/proxy_api/consent/me/vehicles"
	)

	for _, tc := range []struct {
		name            string
		landing         string
		marketing       bool
		callback        string
		marketingStatus int
		portalStatus    int
		wantErr         string
	}{
		{name: "direct", landing: userPage},
		{name: "query and fragment", landing: BaseURL + "/content/euda/gb/en/user.html?next=/error#profile"},
		{name: "relative location", landing: "/content/euda/fr/fr/user.html?locale=fr"},
		{name: "marketing", landing: userPage, marketing: true, callback: identityCallback + "?scopes=openid cars"},
		{name: "marketing relative landing", landing: "/content/euda/de/en/user.html?locale=en", marketing: true, callback: identityCallback},
		{name: "missing callback", marketing: true, wantErr: "marketing consent page is missing callback url"},
		{name: "malformed callback", marketing: true, callback: "%", wantErr: "invalid URL escape"},
		{name: "insecure callback", marketing: true, callback: "http://example.org/callback", wantErr: "unexpected marketing consent callback URL"},
		{name: "http callback", marketing: true, callback: "http://identity.vwgroup.io/oidc/v1/oauth/client/callback", wantErr: "unexpected marketing consent callback URL"},
		{name: "callback wrong path", marketing: true, callback: vwidentity.BaseURL + "/unexpected/callback", wantErr: "unexpected marketing consent callback URL"},
		{name: "marketing callback error", marketing: true, callback: identityCallback, marketingStatus: http.StatusForbidden, wantErr: "403 Forbidden"},
		{name: "portal callback error", landing: userPage, portalStatus: http.StatusUnauthorized, wantErr: "401 Unauthorized"},
		{name: "portal callback error after marketing", landing: userPage, marketing: true, callback: identityCallback, portalStatus: http.StatusForbidden, wantErr: "403 Forbidden"},
		{name: "missing redirect", wantErr: "login redirect"},
		{name: "callback without redirect", portalStatus: http.StatusOK, wantErr: "unexpected landing page"},
		{name: "required consent", landing: BaseURL + "/consent", wantErr: "open the portal and confirm consent"},
		{name: "signin", landing: signin, wantErr: "open the portal and confirm consent"},
		{name: "error page", landing: BaseURL + "/content/euda/de/de/error.html?next=user.html", wantErr: "open the portal and confirm consent"},
		{name: "wrong filename", landing: BaseURL + "/content/euda/de/de/notuser.html", wantErr: "unexpected landing page"},
		{name: "extra segment", landing: userPage + "/extra", wantErr: "unexpected landing page"},
		{name: "wrong prefix", landing: BaseURL + "/content/other/de/de/user.html", wantErr: "unexpected landing page"},
		{name: "wrong host", landing: "https://example.org/content/euda/de/de/user.html", wantErr: "unexpected login redirect host"},
		{name: "insecure landing", landing: "http://" + portalHost + "/content/euda/de/de/user.html", wantErr: "unexpected landing page"},
		{name: "unexpected redirect host", landing: "https://attacker.example/content/euda/de/de/user.html", wantErr: "unexpected login redirect host"},
		{name: "redirect loop", landing: portalCallback, wantErr: "stopped after 10 redirects"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var bodies []*loginTestBody
			var requested []string
			originalRedirectErr := errors.New("original redirect policy")
			v := &API{
				brand:    brands["Audi"],
				user:     "test@example.org",
				password: "test-password",
				Helper: &request.Helper{Client: &http.Client{
					CheckRedirect: func(*http.Request, []*http.Request) error { return originalRedirectErr },
					Transport: loginRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Body != nil {
							defer req.Body.Close()
						}
						uri := req.URL.Scheme + "://" + req.URL.Host + req.URL.Path
						requested = append(requested, uri)
						body := &loginTestBody{Reader: strings.NewReader("")}
						bodies = append(bodies, body)
						resp := &http.Response{
							StatusCode: http.StatusOK,
							Header:     make(http.Header),
							Body:       body,
							Request:    req,
						}
						switch uri {
						case vwidentity.Config.AuthURL:
							resp.StatusCode = http.StatusFound
							resp.Header.Set("Location", signin)
						case signin:
							body.Reader = strings.NewReader(`<form id="emailPasswordForm" action="/login/identifier">
								<input name="_csrf" value="test-csrf"><input name="relayState" value="test-relay"><input name="hmac" value="test-hmac">
							</form>`)
						case identifier:
							assert.Equal(t, http.MethodPost, req.Method)
							require.NoError(t, req.ParseForm())
							assert.Equal(t, "test@example.org", req.PostForm.Get("email"))
							body.Reader = strings.NewReader(`<script>window._IDK = {"csrf_token":"test-csrf","templateModel":{"identifierUrl":"/identifier","postAction":"/authenticate","relayState":"test-relay","hmac":"test-hmac"}};</script>`)
						case authenticate:
							assert.Equal(t, http.MethodPost, req.Method)
							require.NoError(t, req.ParseForm())
							assert.Equal(t, "test-password", req.PostForm.Get("password"))
							resp.StatusCode = http.StatusFound
							resp.Header.Set("Location", portalCallback)
							if tc.marketing {
								resp.Header.Set("Location", marketingPage+"?"+url.Values{"callback": {tc.callback}}.Encode())
							}
						case identityCallback:
							assert.Equal(t, http.MethodGet, req.Method)
							assert.NotContains(t, req.URL.RawQuery, " ")
							resp.StatusCode = http.StatusFound
							if tc.marketingStatus != 0 {
								resp.StatusCode = tc.marketingStatus
							}
							resp.Header.Set("Location", portalCallback)
						case portalCallback:
							assert.Equal(t, http.MethodGet, req.Method)
							resp.StatusCode = http.StatusFound
							if tc.portalStatus != 0 {
								resp.StatusCode = tc.portalStatus
							}
							resp.Header.Set("Location", tc.landing)
							resp.Header.Add("Set-Cookie", "access_token=test-token; Path=/; Secure; HttpOnly")
						case vehicles:
							cookie, err := req.Cookie("access_token")
							require.NoError(t, err)
							assert.Equal(t, "test-token", cookie.Value)
							body.Reader = strings.NewReader(`[]`)
						default:
							assert.False(t, isUserPage(req.URL), "user page must not be fetched")
							assert.NotEqual(t, marketingPage, uri, "marketing page must not be fetched")
						}
						resp.Status = fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
						return resp, nil
					}),
				}},
			}
			v.Client.Jar = &v.jar

			err := v.login(0)
			for _, body := range bodies {
				assert.True(t, body.closed, "login response body must be closed")
			}
			require.NotNil(t, v.Client.CheckRedirect)
			assert.ErrorIs(t, v.Client.CheckRedirect(nil, nil), originalRedirectErr)
			assert.NotContains(t, requested, marketingPage)
			assert.NotContains(t, requested, "https://attacker.example")

			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, requested, portalCallback)
			if tc.marketing {
				callback, err := url.Parse(tc.callback)
				require.NoError(t, err)
				assert.Contains(t, requested, callback.Scheme+"://"+callback.Host+callback.Path)
			}
			_, err = v.Vehicles()
			assert.NoError(t, err)
		})
	}
}
