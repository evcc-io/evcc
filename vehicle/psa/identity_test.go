package psa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestAuthCode(t *testing.T) {
	tc := []struct {
		in, out string
	}{
		{"abc", "abc"},
		{"  abc  ", "abc"},
		{"mymap://oauth2redirect/de?code=abc&scope=openid%20profile", "abc"},
		{"mymap://oauth2redirect/de", "mymap://oauth2redirect/de"},
	}

	for _, tc := range tc {
		assert.Equal(t, tc.out, authCode(tc.in), tc.in)
	}
}

func TestAuthChallenge(t *testing.T) {
	for brand := range brandNames {
		t.Run(brand, func(t *testing.T) {
			config := map[string]any{"user": t.Name() + "@example.com", "country": " FR "}
			ts, err := auth.NewFromConfig(context.Background(), brand, config)
			require.NoError(t, err)
			identity := ts.(*Identity)
			_, err = identity.Token()
			require.Error(t, err)
			assert.Equal(t, api.LoginRequiredError(identity.subject), err)

			challenge, err := identity.StartChallenge()
			require.NoError(t, err)
			assert.Equal(t, api.AuthChallengeCode, challenge.Kind)
			link, err := url.Parse(challenge.Link)
			require.NoError(t, err)
			assert.Equal(t, Oauth2Config(brand, "fr").RedirectURL, link.Query().Get("redirect_uri"))
			assert.Equal(t, "S256", link.Query().Get("code_challenge_method"))
			assert.NotEmpty(t, link.Query().Get("code_challenge"))

			again, err := identity.StartChallenge()
			require.NoError(t, err)
			assert.Equal(t, challenge, again)

			config["country"] = "gb"
			reused, err := auth.NewFromConfig(context.Background(), brand, config)
			require.NoError(t, err)
			assert.Same(t, identity, reused)
			changed, err := identity.StartChallenge()
			require.NoError(t, err)
			assert.NotEqual(t, challenge.Link, changed.Link)
			link, err = url.Parse(changed.Link)
			require.NoError(t, err)
			assert.Equal(t, Oauth2Config(brand, "gb").RedirectURL, link.Query().Get("redirect_uri"))
		})
	}
}

func TestSubmitChallenge(t *testing.T) {
	v := &Identity{
		log: util.NewLogger("test"), ctx: context.Background(),
		brand: "peugeot", country: "de", subject: "test." + t.Name(),
	}
	t.Cleanup(func() { settings.SetString(v.subject, "") })
	_, err := v.SubmitChallenge("code")
	require.EqualError(t, err, "no pending login")
	_, err = v.StartChallenge()
	require.NoError(t, err)

	_, err = v.SubmitChallenge(" ")
	require.EqualError(t, err, "missing authorization code")
	verifier := v.pending.verifier
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, verifier, r.Form.Get("code_verifier"))
		w.Header().Set("Content-Type", "application/json")
		if r.Form.Get("code") != "valid" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","expires_in":3600}`))
	}))
	defer srv.Close()
	v.pending.oc.Endpoint.TokenURL = srv.URL

	_, err = v.SubmitChallenge("invalid")
	require.Error(t, err)
	require.NotNil(t, v.pending)
	challenge, err := v.SubmitChallenge("mymap://oauth2redirect/de?code=valid")
	require.NoError(t, err)
	assert.Nil(t, challenge)
	assert.Nil(t, v.pending)
	assert.True(t, v.Authenticated())
	var token oauth2.Token
	require.NoError(t, settings.Json(v.subject, &token))
	assert.Equal(t, "refresh", token.RefreshToken)
}
