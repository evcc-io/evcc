package idkproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
)

const (
	BaseURL   = "https://idkproxy-service.apps.emea.vwapps.io"
	WellKnown = "https://idkproxy-service.apps.emea.vwapps.io/v1/emea/openid-configuration"
)

var Config = &oidc.ProviderConfig{
	AuthURL:  "https://identity.vwgroup.io/oidc/v1/authorize",
	TokenURL: "https://idkproxy-service.apps.emea.vwapps.io/v1/emea/token",
}

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

// https://github.com/arjenvrh/audi_connect_ha/issues/133

const (
	qmSecret   = "e47866378ef0658ce75d71007a809f34616b9635e2ec228245784c1f63e88d06"
	qmClientId = "c95f4fd2"
)

func qmauth(ts int64) string {
	secret, _ := hex.DecodeString(qmSecret)
	hash := hmac.New(sha256.New, secret)
	hash.Write([]byte(strconv.FormatInt(ts, 10)))
	return hex.EncodeToString(hash.Sum(nil))
}

func qmauthNow() string {
	return "v1:" + qmClientId + ":" + qmauth(time.Now().Unix()/100)
}

// WithStore attaches a persistent store
func (v *Service) WithStore(store store.Store) *Service {
	if store != nil && !reflect.ValueOf(store).IsNil() {
		v.store = store
	}
	return v
}

// Exchange exchanges an VAG identity token for an IDK token
func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := urlvalues.Require(q, "code", "code_verifier"); err != nil {
		return nil, err
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"response_type": {"token id_token"},
		"code":          {q.Get("code")},
		"code_verifier": {q.Get("code_verifier")},
	}

	urlvalues.Merge(data, v.data)

	var res vag.Token

	req, err := request.New(http.MethodPost, Config.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"Accept":       request.JSONContent,
		"x-qmauth":     qmauthNow(),
	})
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// refresh refreshes an IDK token
func (v *Service) refresh(token *vag.Token) (*vag.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"response_type": {"token id_token"},
		"refresh_token": {token.RefreshToken},
	}

	urlvalues.Merge(data, v.data)

	var res vag.Token

	req, err := request.New(http.MethodPost, Config.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"Accept":       request.JSONContent,
		"x-qmauth":     qmauthNow(),
	})
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	// store refreshed token
	if err == nil && v.store != nil {
		fmt.Println("idk refresh remaining:", time.Until(res.Expiry))
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
			fmt.Println("idk load remaining:", time.Until(token.Expiry))
		} else {
			fmt.Println("idk store remaining:", time.Until(token.Expiry))
			// store initial token, typically from code exchange
			_ = v.store.Save(token)
		}
	}

	// don't create tokensource for nil token
	if token == nil {
		return nil
	}

	return vag.RefreshTokenSource(token, v.refresh)
}
