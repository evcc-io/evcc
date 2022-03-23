package aazsproxy

import (
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
	"golang.org/x/oauth2"
)

const BaseURL = "https://aazsproxy-service.apps.emea.vwapps.io"

var Endpoint = &oauth2.Endpoint{
	AuthURL: BaseURL + "/token",
	// TokenURL: BaseURL + "/refresh/v1",
}

type Service struct {
	*request.Helper
}

func New(log *util.Logger) *Service {
	return &Service{
		Helper: request.NewHelper(log),
	}
}

func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := urlvalues.Require(q, "id_token"); err != nil {
		return nil, err
	}

	// TODO make configurable
	data := struct {
		Token     string `json:"token"`
		GrantType string `json:"grant_type"`
		Stage     string `json:"stage"`
		Config    string `json:"config"`
	}{
		Token:     q.Get("id_token"),
		GrantType: "id_token",
		Stage:     "live",
		Config:    "myaudi",
	}

	var res vag.Token

	req, err := request.New(http.MethodPost, Endpoint.AuthURL, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// TODO implement
func (v *Service) Refresh(token *vag.Token) (*vag.Token, error) {
	// req, err := request.New(http.MethodGet, Endpoint.TokenURL, nil, map[string]string{
	// 	"Accept":        "application/json",
	// 	"Authorization": "Bearer " + token.RefreshToken,
	// })

	// var res Token
	// if err == nil {
	// 	err = v.DoJSON(req, &res)
	// }

	// return &res, err

	return nil, api.ErrNotAvailable
}

// RefreshToken implements oauth.TokenRefresher
func (v *Service) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	// TODO implement
	return nil, api.ErrNotAvailable
}

// TokenSource creates a refreshing OAuth2 token source
func (v *Service) TokenSource(token *vag.Token) oauth2.TokenSource {
	return oauth.RefreshTokenSource(&token.Token, v)
}
