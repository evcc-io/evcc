package idkproxy

import (
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

const (
	BaseURL   = "https://idkproxy-service.apps.emea.vwapps.io"
	WellKnown = BaseURL + "/v1/emea/openid-configuration"
)

var Endpoint = &oauth2.Endpoint{
	AuthURL:  vwidentity.Endpoint.AuthURL,
	TokenURL: BaseURL + "/v1/emea/token",
}

type Service struct {
	*request.Helper
}

func New(log *util.Logger) *Service {
	return &Service{
		Helper: request.NewHelper(log),
	}
}

// TODO not implemented
func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := urlvalues.Require(q, "state", "id_token", "access_token", "code"); err != nil {
		return nil, err
	}

	data := map[string]string{
		"region":            "emea",
		"redirect_uri":      "weconnect://authenticated",
		"state":             q.Get("state"),
		"id_token":          q.Get("id_token"),
		"access_token":      q.Get("access_token"),
		"authorizationCode": q.Get("code"),
	}

	var res vag.Token

	req, err := request.New(http.MethodPost, Endpoint.AuthURL, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

func (v *Service) Refresh(token *vag.Token) (*vag.Token, error) {
	req, err := request.New(http.MethodGet, Endpoint.TokenURL, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + token.RefreshToken,
	})

	var res vag.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Service) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	return nil, api.ErrNotAvailable
}

// TokenSource creates a refreshing OAuth2 token source
func (v *Service) TokenSource(token *vag.Token) oauth2.TokenSource {
	return nil
}
