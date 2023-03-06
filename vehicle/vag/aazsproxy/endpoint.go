package aazsproxy

import (
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/cariad"
	"golang.org/x/oauth2"
)

var Endpoint = &oauth2.Endpoint{
	AuthURL: cariad.BaseURL + "/login/v1/audi/token",
}

type Service struct {
	*request.Helper
}

func New(log *util.Logger) *Service {
	return &Service{
		Helper: request.NewHelper(log),
	}
}

// Exchange exchanges an VAG identity or IDK token for an AAZS token
func (v *Service) Exchange(config, token string) (*vag.Token, error) {
	data := struct {
		Token     string `json:"token"`
		GrantType string `json:"grant_type"`
		Stage     string `json:"stage"`
		Config    string `json:"config"`
	}{
		Token:     token,
		GrantType: "id_token",
		Stage:     "live",
		Config:    config,
	}

	var res vag.Token

	req, err := request.New(http.MethodPost, Endpoint.AuthURL, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// TokenSource creates token source. Token is NOT refreshed but will expire.
func (v *Service) TokenSource(token *vag.Token) vag.TokenSource {
	return vag.RefreshTokenSource(token, nil)
}
