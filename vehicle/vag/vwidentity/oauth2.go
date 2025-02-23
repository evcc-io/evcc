package vwidentity

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
	"golang.org/x/oauth2"
)

// Login performs VW identity login with optional code challenge
func Oauth2Login(log *util.Logger, oc *oauth2.Config, user, password string) (vag.TokenSource, error) {
	oc.Endpoint = Config.NewProvider(context.Background()).Endpoint()

	// add code challenge
	cv := oauth2.GenerateVerifier()
	q := url.Values{
		"response_type":         {"code id_token"},
		"client_id":             {oc.ClientID},
		"redirect_uri":          {oc.RedirectURL},
		"code_challenge_method": {"S256"},
		"code_challenge":        {oauth2.S256ChallengeFromVerifier(cv)},
		"scope":                 {strings.Join(oc.Scopes, " ")},
	}

	uri := fmt.Sprintf("%s?%s", oc.Endpoint.AuthURL, q.Encode())

	vwi := New(log)
	q, err := vwi.Login(uri, user, password)
	if err != nil {
		return nil, err
	}

	if err := urlvalues.Require(q, "id_token", "code"); err != nil {
		return nil, err
	}

	v := url.Values{
		"client_id":     {oc.ClientID},
		"client_secret": {oc.ClientSecret},
		"redirect_uri":  {oc.RedirectURL},
		"grant_type":    {"authorization_code"},
		"code":          {q.Get("code")},
		"code_verifier": {cv},
	}

	os := &Oauth2Service{Helper: request.NewHelper(log), Config: oc}

	token, err := os.Token(v)
	if err != nil {
		return nil, err
	}

	return os.TokenSource(token), nil
}

type Oauth2Service struct {
	*oauth2.Config
	*request.Helper
}

func (v *Oauth2Service) Token(data url.Values) (*vag.Token, error) {
	var res vag.Token

	req, err := request.New(http.MethodPost, v.Endpoint.TokenURL, strings.NewReader(data.Encode()), request.URLEncoding, request.AcceptJSON)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

func (v *Oauth2Service) Refresh(token *vag.Token) (*vag.Token, error) {
	data := url.Values{
		"client_id":     {v.ClientID},
		"client_secret": {v.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	return v.Token(data)
}

// TokenSource creates token source. Token is refreshed automatically.
func (v *Oauth2Service) TokenSource(token *vag.Token) vag.TokenSource {
	return vag.RefreshTokenSource(token, v.Refresh)
}
