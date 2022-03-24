package aazsproxy

import (
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
	"golang.org/x/oauth2"
)

const BaseURL = "https://aazsproxy-service.apps.emea.vwapps.io"

var Endpoint = &oauth2.Endpoint{
	AuthURL: BaseURL + "/token",
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

// TokenSource creates token source. Token is NOT refreshed but will expire.
func (v *Service) TokenSource(token *vag.Token) vag.TokenSource {
	return vag.RefreshTokenSource(token, nil)
}
