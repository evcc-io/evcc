package rabot

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type tokenSource struct {
	*request.Helper
	log             *util.Logger
	login, password string
}

func TokenSource(log *util.Logger, login, password string) (oauth2.TokenSource, error) {
	c := &tokenSource{
		Helper:   request.NewHelper(log),
		log:      log,
		login:    login,
		password: password,
	}

	token, err := c.authenticate()
	if err != nil {
		return nil, err
	}

	return oauth.RefreshTokenSource(token, c.refreshToken), nil
}

func (c *tokenSource) authenticate() (*oauth2.Token, error) {
	data := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{
		Login:    c.login,
		Password: c.password,
	}

	req, err := request.New(http.MethodPost, BaseURI+"/api/prosumer/session/login", request.MarshalJSON(data), request.JSONEncoding, map[string]string{
		"Authorization": "Bearer " + AppToken,
	})
	if err != nil {
		return nil, err
	}

	var res LoginResponse
	if err := c.DoJSON(req, &res); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	c.log.Redact(res.SessionToken, res.RefreshToken)

	return &oauth2.Token{
		AccessToken:  res.SessionToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Add(time.Second * time.Duration(res.ExpiresIn)),
	}, nil
}

func (c *tokenSource) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := struct {
		RefreshToken string `json:"refreshToken"`
	}{
		RefreshToken: token.RefreshToken,
	}

	req, err := request.New(http.MethodPost, BaseURI+"/api/prosumer/session/refresh", request.MarshalJSON(data), request.JSONEncoding, map[string]string{
		"Authorization": "Bearer " + AppToken,
	})
	if err != nil {
		return nil, err
	}

	var res LoginResponse
	if err := c.DoJSON(req, &res); err != nil {
		// fall back to full re-authentication
		return c.authenticate()
	}

	c.log.Redact(res.SessionToken, res.RefreshToken)

	return &oauth2.Token{
		AccessToken:  res.SessionToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Add(time.Second * time.Duration(res.ExpiresIn)),
	}, nil
}
