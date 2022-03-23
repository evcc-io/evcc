package mbb

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
	"golang.org/x/oauth2"
)

const (
	BaseURL  = "https://mbboauth-1d.prd.ece.vwg-connect.com"
	TokenURL = BaseURL + "/mbbcoauth/mobile/oauth2/v1/token"
)

type Service struct {
	*request.Helper
	clientID string
}

func New(log *util.Logger, clientID string) *Service {
	return &Service{
		Helper:   request.NewHelper(log),
		clientID: clientID,
	}
}

func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := urlvalues.Require(q, "id_token"); err != nil {
		return nil, err
	}

	data := url.Values{
		"grant_type": {"id_token"},
		"token":      {q.Get("id_token")},
		"scope":      {"sc2:fal"},
	}

	req, err := request.New(http.MethodPost, TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-Client-Id":  v.clientID,
	})

	var res vag.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	// check if token response contained error
	if errT := res.Error(); err != nil && errT != nil {
		err = fmt.Errorf("token exchange: %w", errT)
	}

	return &res, err
}

func (v *Service) Refresh(token *vag.Token) (*vag.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"scope":         {"sc2:fal"},
	}

	req, err := request.New(http.MethodPost, TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-Client-Id":  v.clientID,
	})

	var res vag.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Service) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	res, err := v.Refresh(&vag.Token{
		Token: *token,
	})

	if err != nil {
		return nil, err
	}

	return &res.Token, err
}

// TokenSource creates a refreshing OAuth2 token source
func (v *Service) TokenSource(token *vag.Token) oauth2.TokenSource {
	return oauth.RefreshTokenSource(&token.Token, v)
}
