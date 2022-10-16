package aazsproxy

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag"
	"golang.org/x/oauth2"
)

const BaseURL = "https://aazsproxy-service.apps.emea.vwapps.io"

var Endpoint = &oauth2.Endpoint{
	AuthURL: BaseURL + "/token",
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
	if v.store != nil {
		if token == nil {
			if err := v.store.Load(&token); err != nil || token == nil {
				return nil
			}
			fmt.Println("azs load remaining:", time.Until(token.Expiry))
		} else {
			fmt.Println("azs store remaining:", time.Until(token.Expiry))
			// store initial token, typically from code exchange
			_ = v.store.Save(token)
		}
	}

	// don't create tokensource for nil token
	if token == nil {
		return nil
	}

	return vag.RefreshTokenSource(token, nil)
}
