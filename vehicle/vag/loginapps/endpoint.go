package loginapps

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag/cariad"
	"golang.org/x/oauth2"
)

var Endpoint = &oauth2.Endpoint{
	AuthURL:  cariad.BaseURL + "/user-login/login/v1",
	TokenURL: cariad.BaseURL + "/user-login/refresh/v1",
}

type Service struct {
	*request.Helper
}

func New(log *util.Logger) *Service {
	return &Service{
		Helper: request.NewHelper(log),
	}
}

func (v *Service) Exchange(q url.Values) (*oauth2.Token, error) {
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

	var res oauth2.Token

	req, err := request.New(http.MethodPost, Endpoint.AuthURL, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Service) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	body := url.Values{
		"grant_type":    []string{"refresh_token"},
		"refresh_token": []string{token.RefreshToken},
		"client_id":     []string{cariad.ClientID},
	}

	req, err := request.New(
		http.MethodPost,
		cariad.BaseURL+"/login/v1/idk/token",
		strings.NewReader(body.Encode()),
		request.URLEncoding,
		map[string]string{
			"Connection":             "keep-alive",
			"User-Agent":             cariad.UserAgent,
			"Accept":                 "application/json",
			"x-android-package-name": cariad.AndroidPackageName,
		},
	)

	var res oauth2.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// TokenSource creates a refreshing oauth2 token source
func (v *Service) TokenSource(token *oauth2.Token) oauth2.TokenSource {
	return oauth.RefreshTokenSource(token, v)
}
