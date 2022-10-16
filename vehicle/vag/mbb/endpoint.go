package mbb

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
)

const (
	BaseURL  = "https://mbboauth-1d.prd.ece.vwg-connect.com"
	TokenURL = BaseURL + "/mbbcoauth/mobile/oauth2/v1/token"
)

var _ vag.TokenExchanger = (*Service)(nil)

type Service struct {
	*request.Helper
	clientID string
	store    store.Store
}

func New(log *util.Logger, clientID string) *Service {
	return &Service{
		Helper:   request.NewHelper(log),
		clientID: clientID,
	}
}

// WithStore attaches a persistent store
func (v *Service) WithStore(store store.Store) *Service {
	if store != nil && !reflect.ValueOf(store).IsNil() {
		v.store = store
	}
	return v
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

func (v *Service) refresh(token *vag.Token) (*vag.Token, error) {
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

	// store refreshed token
	if err == nil && v.store != nil {
		fmt.Println("mbb refresh remaining:", time.Until(res.Expiry))
		err = v.store.Save(res)
	}

	return &res, err
}

// TokenSource creates token source. Token is refreshed automatically.
func (v *Service) TokenSource(token *vag.Token) vag.TokenSource {
	if v.store != nil {
		if token == nil {
			if err := v.store.Load(&token); err != nil || token == nil {
				return nil
			}
			fmt.Println("mbb load remaining:", time.Until(token.Expiry))
		} else {
			// store initial token, typically from code exchange
			fmt.Println("mbb store remaining:", time.Until(token.Expiry))
			_ = v.store.Save(token)
		}
	}

	// don't create tokensource for nil token
	if token == nil {
		return nil
	}

	return vag.RefreshTokenSource(token, v.refresh)
}
