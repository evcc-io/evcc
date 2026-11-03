package vwidentity

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

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
