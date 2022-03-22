package tokenrefreshservice

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag"
	"golang.org/x/oauth2"
)

const (
	BaseURL         = "https://tokenrefreshservice.apps.emea.vwapps.io"
	CodeExchangeURL = BaseURL + "/exchangeAuthCode"
	RefreshTokenURL = BaseURL + "/refreshTokens"
)

type Service struct {
	*request.Helper
}

func New(log *util.Logger) *Service {
	return &Service{
		Helper: request.NewHelper(log),
	}
}

func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := util.RequireValues(q, "id_token", "code"); err != nil {
		return nil, err
	}

	data := url.Values{
		"auth_code": {q.Get("code")},
		"id_token":  {q.Get("id_token")},
		"brand":     {"skoda"}, // TODO parametrize
	}

	for k, v := range q {
		data[k] = v
	}

	var res vag.Token

	req, err := request.New(http.MethodPost, CodeExchangeURL, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

func (v *Service) Refresh(token *vag.Token, q url.Values) (*vag.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"brand":         {"skoda"}, // TODO parametrize
	}

	for k, v := range q {
		data[k] = v
	}

	var res vag.Token

	req, err := request.New(http.MethodPost, RefreshTokenURL, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Service) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	res, err := v.Refresh(&vag.Token{
		Token: *token,
	}, nil)

	if err != nil {
		return nil, err
	}

	return &res.Token, err
}

// TokenSource creates a refreshing OAuth2 token source
func (v *Service) TokenSource(token *vag.Token) oauth2.TokenSource {
	return oauth.RefreshTokenSource(&token.Token, v)
}
