package loginapps

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"golang.org/x/oauth2"
)

const (
	BaseURL = "https://login.apps.emea.vwapps.io"
)

var Endpoint = &oauth2.Endpoint{
	AuthURL:  BaseURL + "/login/v1",
	TokenURL: BaseURL + "/refresh/v1",
}

type Service struct {
	*request.Helper
	store store.Store
}

func New(log *util.Logger) *Service {
	return &Service{
		Helper: request.NewHelper(log),
	}
}

// WithStore attaches a persistent store
func (v *Service) WithStore(store store.Store) *Service {
	if store != nil && !reflect.ValueOf(store).IsNil() {
		v.store = store
	}
	return v
}

func (v *Service) Exchange(q url.Values) (*Token, error) {
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

	var res Token

	req, err := request.New(http.MethodPost, Endpoint.AuthURL, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

func (v *Service) refresh(token *Token) (*Token, error) {
	req, err := request.New(http.MethodGet, Endpoint.TokenURL, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + token.RefreshToken,
	})

	var res Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	// store refreshed token
	if err == nil && v.store != nil {
		fmt.Println("loginapps refresh remaining:", time.Until(res.Expiry))
		err = v.store.Save(res)
	}

	return &res, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Service) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	res, err := v.refresh((*Token)(token))
	return (*oauth2.Token)(res), err
}

// TokenSource creates a refreshing oauth2 token source
func (v *Service) TokenSource(token *Token) oauth2.TokenSource {
	if v.store != nil {
		if token == nil {
			if err := v.store.Load(&token); err != nil || token == nil {
				return nil
			}
			fmt.Println("loginapps load remaining:", time.Until(token.Expiry))
		} else {
			fmt.Println("loginapps store remaining:", time.Until(token.Expiry))
			// store initial token, typically from code exchange
			_ = v.store.Save(token)
		}
	}

	// don't create tokensource for nil token
	if token == nil {
		return nil
	}

	return oauth.RefreshTokenSource((*oauth2.Token)(token), v)
}
