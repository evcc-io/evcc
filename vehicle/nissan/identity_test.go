package nissan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const loginPage = `<html><body>
<form method="post" action="/unrelated"><input type="hidden" name="foo" value="1"/></form>
<form id="loginForm" method="post" action="../../commonauth">
	<input type="hidden" name="sessionDataKey" value="abc-123"/>
	<input type="hidden" name="regionCode" value="EU"/>
	<input type="text" name="username" value=""/>
	<input type="password" name="password"/>
</form>
</body></html>`

func TestLoginForm(t *testing.T) {
	form, err := loginForm(strings.NewReader(loginPage))
	require.NoError(t, err)

	assert.Equal(t, "../../commonauth", form.Action)
	assert.Equal(t, "abc-123", form.Inputs["sessionDataKey"])
	assert.Equal(t, "EU", form.Inputs["regionCode"])
	assert.NotContains(t, form.Inputs, "foo")
}

func TestLoginFormMissing(t *testing.T) {
	_, err := loginForm(strings.NewReader(`<html><body><form action="/x"><input name="password"/></form></body></html>`))
	assert.Error(t, err)
}

func TestTokenConversion(t *testing.T) {
	_, err := (&Token{Error: "invalid_grant", ErrorDescription: "expired"}).Token()
	assert.Error(t, err)

	_, err = (&Token{}).Token()
	assert.Error(t, err)

	token, err := (&Token{AccessToken: "a"}).Token()
	require.NoError(t, err)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.False(t, token.Expiry.IsZero(), "missing expires_in must not yield a non-expiring token")
}

func response(r *http.Request, code int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Request:    r,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": {contentType}},
	}
}

func redirect(r *http.Request, location string) *http.Response {
	res := response(r, http.StatusFound, "text/html", "")
	res.Header.Set("Location", location)
	return res
}

// testIdentity returns an identity talking to a fake MyNISSAN OneID and Kamereon backend
func testIdentity(t *testing.T, password string) *Identity {
	t.Helper()

	v := &Identity{
		Helper:   request.NewHelper(util.NewLogger("test")),
		log:      util.NewLogger("test"),
		user:     "user",
		password: "pass",
	}

	var state string

	v.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch path := r.URL.Host + r.URL.Path; path {
		case authBaseURI.Host + "/oauth2/authorize":
			q := r.URL.Query()
			assert.Equal(t, "code", q.Get("response_type"))
			assert.Equal(t, ClientID, q.Get("client_id"))
			assert.Equal(t, RedirectURI, q.Get("redirect_uri"))
			assert.Equal(t, "S256", q.Get("code_challenge_method"))
			assert.NotEmpty(t, q.Get("code_challenge"))
			assert.Equal(t, Locale, q.Get("locale"))
			assert.Equal(t, AuthBrand, q.Get("brand"))
			assert.Equal(t, AuthClient, q.Get("client"))
			state = q.Get("state")

			// the login page is reached through an internal redirect
			return redirect(r, AuthBaseURL+"/authenticationendpoint/login.do"), nil

		case authBaseURI.Host + "/authenticationendpoint/login.do":
			res := response(r, http.StatusOK, "text/html", loginPage)
			res.Header.Set("Set-Cookie", "JSESSIONID=session; Path=/")
			return res, nil

		case authBaseURI.Host + "/commonauth":
			// the session cookie set with the login page must be returned
			assert.Equal(t, "JSESSIONID=session", r.Header.Get("Cookie"))
			assert.Equal(t, AuthBaseURL, r.Header.Get("Origin"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			form, err := url.ParseQuery(string(body))
			require.NoError(t, err)

			assert.Equal(t, "abc-123", form.Get("sessionDataKey"))
			assert.Equal(t, "user", form.Get("userName"))
			assert.Equal(t, "EU/user", form.Get("username"))

			// rejected credentials re-render the login form
			if form.Get("password") != password {
				return response(r, http.StatusOK, "text/html", loginPage), nil
			}

			return redirect(r, fmt.Sprintf("%s?code=authcode&state=%s", RedirectURI, url.QueryEscape(state))), nil

		case authBaseURI.Host + "/oauth2/token":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			form, err := url.ParseQuery(string(body))
			require.NoError(t, err)

			assert.Equal(t, "authorization_code", form.Get("grant_type"))
			assert.Equal(t, "authcode", form.Get("code"))
			assert.NotEmpty(t, form.Get("code_verifier"))
			assert.Empty(t, form.Get("client_secret"))

			return response(r, http.StatusOK, request.JSONContent,
				`{"access_token":"wso2-access","id_token":"wso2-id","token_type":"Bearer","expires_in":3600}`), nil

		case strings.TrimPrefix(UserBaseURL, "https://") + "/v1/oauth2/access_token":
			assert.Equal(t, "wso2-id", r.Header.Get("Authorization"))
			assert.Equal(t, KamereonPlatform, r.URL.Query().Get("platform"))

			return response(r, http.StatusOK, request.JSONContent,
				`{"access_token":"kamereon-access","refresh_token":"kamereon-refresh","token_type":"Bearer","expires_in":900}`), nil

		case strings.TrimPrefix(UserBaseURL, "https://") + "/v1/oauth2/refresh-token":
			assert.Equal(t, "kamereon-refresh", r.Header.Get("Authorization"))

			var data struct{ Scope string }
			require.NoError(t, json.NewDecoder(r.Body).Decode(&data))
			assert.Equal(t, KamereonScope, data.Scope)

			return response(r, http.StatusOK, request.JSONContent,
				`{"access_token":"refreshed-access","token_type":"Bearer","expires_in":900}`), nil

		default:
			return nil, fmt.Errorf("unexpected request: %s", path)
		}
	})

	return v
}

func TestLogin(t *testing.T) {
	v := testIdentity(t, "pass")

	token, err := v.login()
	require.NoError(t, err)
	assert.Equal(t, "kamereon-access", token.AccessToken)
	assert.Equal(t, "kamereon-refresh", token.RefreshToken)

	// login must not leave the cookie jar or redirect handler behind
	assert.Nil(t, v.Client.Jar)
	assert.Nil(t, v.Client.CheckRedirect)
}

func TestLoginInvalidCredentials(t *testing.T) {
	v := testIdentity(t, "other")

	_, err := v.login()
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestRefresh(t *testing.T) {
	v := testIdentity(t, "pass")

	token, err := v.refreshKamereonToken("kamereon-refresh")
	require.NoError(t, err)
	assert.Equal(t, "refreshed-access", token.AccessToken)
	assert.False(t, token.Expiry.IsZero())
}

func TestTokenErrorMessage(t *testing.T) {
	// the api error description must survive a non-2xx status
	v := testIdentity(t, "pass")
	v.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return response(r, http.StatusUnauthorized, request.JSONContent,
			`{"error":"invalid_token","error_description":"id token has expired"}`), nil
	})

	_, err := v.kamereonToken("wso2-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id token has expired")
	assert.Contains(t, err.Error(), "401")
}

func TestLoginFormNotFound(t *testing.T) {
	v := testIdentity(t, "pass")
	v.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return response(r, http.StatusOK, "text/html", "<html><body>maintenance</body></html>"), nil
	})

	_, err := v.login()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login form not found")
}
