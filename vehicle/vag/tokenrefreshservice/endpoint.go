package tokenrefreshservice

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
	BaseURL         = "https://tokenrefreshservice.apps.emea.vwapps.io"
	CodeExchangeURL = BaseURL + "/exchangeAuthCode"
	RefreshTokenURL = BaseURL + "/refreshTokens"
)

var _ vag.TokenExchanger = (*Service)(nil)

type Service struct {
	*request.Helper
	data  url.Values
	store store.Store
}

func New(log *util.Logger, q url.Values) *Service {
	return &Service{
		Helper: request.NewHelper(log),
		data:   q,
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
	if err := urlvalues.Require(q, "id_token", "code"); err != nil {
		return nil, err
	}

	data := url.Values{
		"auth_code": {q.Get("code")},
		"id_token":  {q.Get("id_token")},
	}

	urlvalues.Merge(data, v.data, q)

	var res vag.Token

	req, err := request.New(http.MethodPost, CodeExchangeURL, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

func (v *Service) refresh(token *vag.Token) (*vag.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	urlvalues.Merge(data, v.data)

	var res vag.Token

	req, err := request.New(http.MethodPost, RefreshTokenURL, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	// store refreshed token
	if err == nil && v.store != nil {
		fmt.Println("trs refresh remaining:", time.Until(res.Expiry))
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
			fmt.Println("trs load remaining:", time.Until(token.Expiry))
		} else {
			// store initial token, typically from code exchange
			fmt.Println("trs store remaining:", time.Until(token.Expiry))
			_ = v.store.Save(token)
		}
	}

	return vag.RefreshTokenSource(token, v.refresh)
}
