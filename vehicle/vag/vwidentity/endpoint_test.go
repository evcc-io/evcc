package vwidentity

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type formPost struct {
	url  string
	form url.Values
}

// recorder answers every request with the next body and records the posted forms
type recorder struct {
	bodies []string
	posts  []formPost
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(req.Body)
	form, _ := url.ParseQuery(string(b))
	r.posts = append(r.posts, formPost{req.URL.String(), form})

	body := r.bodies[0]
	r.bodies = r.bodies[1:]

	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func TestSubmitCredentialsLegacy(t *testing.T) {
	signin := `<form id="emailPasswordForm" action="/signin-service/v1/client/login/identifier">
		<input type="hidden" name="_csrf" value="c1">
		<input type="hidden" name="relayState" value="r1">
		<input type="hidden" name="hmac" value="h1">
	</form>`

	credentials := `<script>window._IDK = {
		templateModel: {"hmac":"h2","relayState":"r2","postAction":"login/authenticate","identifierUrl":"login/identifier","error":null},
		csrf_token: 'c2'
	}</script>`

	rec := &recorder{bodies: []string{credentials, ""}}
	resp, err := SubmitCredentials(&http.Client{Transport: rec}, []byte(signin), "user", "pass")
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, rec.posts, 2)
	assert.Equal(t, BaseURL+"/signin-service/v1/client/login/identifier", rec.posts[0].url)
	assert.Equal(t, url.Values{"_csrf": {"c1"}, "relayState": {"r1"}, "hmac": {"h1"}, "email": {"user"}}, rec.posts[0].form)
	assert.Equal(t, BaseURL+"/signin-service/v1/client/login/authenticate", rec.posts[1].url)
	assert.Equal(t, url.Values{"_csrf": {"c2"}, "relayState": {"r2"}, "hmac": {"h2"}, "email": {"user"}, "password": {"pass"}}, rec.posts[1].form)
}

func TestSubmitCredentialsUniversal(t *testing.T) {
	signin := `<form method="POST"><input type="hidden" name="state" value="s1"></form>`

	rec := &recorder{bodies: []string{""}}
	resp, err := SubmitCredentials(&http.Client{Transport: rec}, []byte(signin), "user", "pass")
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, rec.posts, 1)
	assert.Equal(t, BaseURL+"/u/login?state=s1", rec.posts[0].url)
	assert.Equal(t, url.Values{"state": {"s1"}, "username": {"user"}, "password": {"pass"}}, rec.posts[0].form)
}

func TestAuthLocationAuth0MarketingConsent(t *testing.T) {
	page, err := url.Parse(BaseURL + "/v2/login/ui/marketing-consent?x=1")
	require.NoError(t, err)

	var posted url.Values
	v := &Service{Helper: &request.Helper{Client: &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport: loginRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, BaseURL+"/v2/login/ui/marketing-consent/skip", req.URL.String())
			require.NoError(t, req.ParseForm())
			posted = req.PostForm
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": {"weconnect://authenticated?code=c1"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}),
	}}}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`<script>window._IDK = {"Data":{"channels":[{"channelId":"MAIL","channelType":"BOUND_TO_BASIC_AGREEMENT"},{"channelId":"APP","channelType":"NOT_USED"}],"state":"s1","relayStateToken":"r1","mdKey":"k1","mdLanguage":"en"}};</script>`)),
		Request:    &http.Request{URL: page},
	}

	q, err := v.authLocation(resp)
	require.NoError(t, err)
	assert.Equal(t, "c1", q.Get("code"))
	assert.Equal(t, url.Values{
		"channelMAIL": {"true"}, "channelAPP": {"not_used"},
		"state": {"s1"}, "relayState": {"r1"}, "mdKey": {"k1"}, "mdLanguage": {"en"},
	}, posted)
}

type loginRoundTripFunc func(*http.Request) (*http.Response, error)

func (f loginRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
