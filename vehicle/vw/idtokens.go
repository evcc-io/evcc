package vw

import (
	"net/http"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// idToken is the non-OIDC compliant VW ID token structure
type idToken struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	Expiry       time.Time
}

func (t *idToken) TokenSource(log *util.Logger) oauth2.TokenSource {
	return &idTokenSource{
		Helper: request.NewHelper(log),
	}
}

type idTokenSource struct {
	token idToken
	*request.Helper
}

func (ts *idTokenSource) Token() (*oauth2.Token, error) {
	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		err = ts.RefreshToken()
	}

	ot := &oauth2.Token{
		AccessToken: ts.token.AccessToken,
		Expiry:      ts.token.Expiry,
	}

	return ot, err
}

func (ts *idTokenSource) RefreshToken() error {
	uri := "https://login.apps.emea.vwapps.io/refresh/v1"

	headers := map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + ts.token.RefreshToken,
	}

	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err == nil {
		var tokens idToken
		if err = ts.DoJSON(req, &tokens); err == nil {
			ts.token = tokens
		}
	}

	return err
}
