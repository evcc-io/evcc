package psa

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// Challenge implements api.InteractiveAuthProvider: the login form.
func (o *Identity) Challenge() *api.AuthChallenge {
	return &api.AuthChallenge{
		Fields: []api.AuthField{
			{Name: "user", Type: "email"},
			{Name: "password", Type: "password"},
		},
	}
}

// Submit implements api.InteractiveAuthProvider. PSA uses the OAuth resource
// owner password grant, so a single request obtains the token - no captcha or
// browser redirect is involved, hence no follow-up challenge.
func (o *Identity) Submit(values map[string]string) (*api.AuthChallenge, error) {
	token, err := o.passwordToken(values["user"], values["password"])
	if err != nil {
		return nil, err
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.update(token)
	return nil, nil
}

// passwordToken performs the OAuth resource owner password credentials grant
// against the brand token endpoint, authenticating the client via HTTP Basic
// and scoping the login to the brand realm.
func (o *Identity) passwordToken(user, password string) (*oauth2.Token, error) {
	data := url.Values{
		"grant_type": {"password"},
		"username":   {user},
		"password":   {password},
		"scope":      {strings.Join(o.oc.Scopes, " ")},
		"realm":      {o.realm},
	}

	req, err := request.New(http.MethodPost, o.oc.Endpoint.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  request.FormContent,
		"Accept":        request.JSONContent,
		"Authorization": transport.BasicAuthHeader(o.oc.ClientID, o.oc.ClientSecret),
	})
	if err != nil {
		return nil, err
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := request.NewHelper(o.log).DoJSON(req, &res); err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		TokenType:    res.TokenType,
	}
	if res.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)
	}
	return token, nil
}
